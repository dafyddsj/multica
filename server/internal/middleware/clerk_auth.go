package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/multica-ai/multica/server/internal/clerk"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

var machineTokenPrefixes = []string{
	"mat_",
	"mul_",
	"mcn_",
	"mdt_",
	"mpi_",
	"mpc_",
}

// Authenticate is the Redis-style composition point: a nil Clerk client
// is today's native Auth middleware. When Clerk is configured, human
// sessions must be Clerk JWTs; machine prefixes still fall through to
// native so daemons, PATs, and task tokens keep working.
func Authenticate(queries *db.Queries, patCache *auth.PATCache, cloudPAT *auth.CloudPATVerifier, clerkClient *clerk.Client) func(http.Handler) http.Handler {
	native := Auth(queries, patCache, cloudPAT)
	if clerkClient == nil {
		return native
	}
	return clerkThenNative(clerkClient, queries, native)
}

// HideNativeAuth 404s email/Google login when Clerk is the control plane.
// The native handlers stay mounted so upstream merges keep applying; this
// wrapper is the only behavior change.
func HideNativeAuth(clerkEnabled bool) func(http.Handler) http.Handler {
	if !clerkEnabled {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"native auth disabled"}`, http.StatusNotFound)
		})
	}
}

func clerkThenNative(clerkClient *clerk.Client, store clerk.UserStore, native func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		nativeHandler := native(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Del("X-Actor-Source")

			tokenString, fromCookie := extractToken(r)
			if tokenString == "" {
				slog.Debug("auth: no token found", "path", r.URL.Path)
				http.Error(w, `{"error":"missing authorization"}`, http.StatusUnauthorized)
				return
			}

			if isMachineToken(tokenString) {
				nativeHandler.ServeHTTP(w, r)
				return
			}

			// Leftover native cookies are not a Clerk session.
			if fromCookie {
				slog.Debug("auth: native cookie rejected while clerk is enabled", "path", r.URL.Path)
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			identity, err := clerkClient.Resolve(r.Context(), tokenString, store)
			if err != nil {
				if isNativeHMACBearer(tokenString) {
					nativeHandler.ServeHTTP(w, r)
					return
				}
				slog.Warn("auth: clerk session rejected", "path", r.URL.Path, "error", err)
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			if rejectTemporarilyDisabledUser(w, r, identity.UserID, identity.Email, "clerk") {
				return
			}
			r.Header.Set("X-User-ID", identity.UserID)
			if identity.Email != "" {
				r.Header.Set("X-User-Email", identity.Email)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isMachineToken(token string) bool {
	for _, prefix := range machineTokenPrefixes {
		if strings.HasPrefix(token, prefix) {
			return true
		}
	}
	return false
}

func isNativeHMACBearer(token string) bool {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return auth.JWTSecret(), nil
	})
	return err == nil && parsed != nil && parsed.Valid
}
