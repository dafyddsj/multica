package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/multica-ai/multica/server/internal/clerk"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type testVerifier struct {
	id  string
	err error
}

func (t testVerifier) Verify(context.Context, string) (string, error) {
	return t.id, t.err
}

type mappedUserStore struct {
	user db.User
}

func (m mappedUserStore) GetUserByClerkID(_ context.Context, _ pgtype.Text) (db.User, error) {
	return m.user, nil
}

func (mappedUserStore) GetUserByEmail(context.Context, string) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}

func (mappedUserStore) CreateUser(context.Context, db.CreateUserParams) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}

func (mappedUserStore) BindUserClerkID(context.Context, db.BindUserClerkIDParams) (db.User, error) {
	return db.User{}, pgx.ErrNoRows
}

func TestAuthenticateNilClerkUsesNativeJWT(t *testing.T) {
	handler := Authenticate(nil, nil, nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-User-ID"); got != "test-user-id" {
			t.Fatalf("X-User-ID: got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+generateToken(validClaims(), auth.JWTSecret()))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("native JWT: want 204, got %d %s", w.Code, w.Body.String())
	}
}

func TestAuthenticateClerkAcceptsNativeHMACBearer(t *testing.T) {
	client := &clerk.Client{Verifier: testVerifier{err: clerk.ErrInvalidSession}}
	handler := clerkThenNative(client, mappedUserStore{}, Auth(nil, nil, nil))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-User-ID"); got != "test-user-id" {
			t.Fatalf("X-User-ID: got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+generateToken(validClaims(), auth.JWTSecret()))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("cli HMAC bearer: want 204, got %d %s", w.Code, w.Body.String())
	}
}

func TestAuthenticateClerkRejectsNativeCookie(t *testing.T) {
	client := &clerk.Client{Verifier: testVerifier{err: clerk.ErrInvalidSession}}
	handler := clerkThenNative(client, mappedUserStore{}, Auth(nil, nil, nil))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next must not run for a native cookie when Clerk is on")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.AuthCookieName, Value: generateToken(validClaims(), auth.JWTSecret())})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d %s", w.Code, w.Body.String())
	}
}

func TestAuthenticateClerkAcceptsMappedSession(t *testing.T) {
	user := db.User{
		ID:    util.MustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Email: "ada@example.com",
		Name:  "Ada",
	}
	client := &clerk.Client{Verifier: testVerifier{id: "user_clerk"}}
	handler := clerkThenNative(client, mappedUserStore{user: user}, Auth(nil, nil, nil))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-User-ID"); got != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
			t.Fatalf("X-User-ID: got %q", got)
		}
		if got := r.Header.Get("X-User-Email"); got != "ada@example.com" {
			t.Fatalf("X-User-Email: got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer clerk-session")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("clerk session: want 204, got %d %s", w.Code, w.Body.String())
	}
}

func TestAuthenticateClerkDelegatesMachineTokens(t *testing.T) {
	client := &clerk.Client{Verifier: testVerifier{id: "should-not-run"}}
	handler := clerkThenNative(client, mappedUserStore{}, Auth(nil, nil, nil))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("native PAT path with nil queries must not reach next")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer mul_not-a-real-pat")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("machine token should hit native and 401, got %d %s", w.Code, w.Body.String())
	}
}

func TestHideNativeAuth(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	off := HideNativeAuth(false)(next)
	req := httptest.NewRequest(http.MethodPost, "/auth/send-code", nil)
	w := httptest.NewRecorder()
	off.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("clerk off: want 200, got %d", w.Code)
	}

	on := HideNativeAuth(true)(next)
	w = httptest.NewRecorder()
	on.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("clerk on: want 404, got %d", w.Code)
	}
}
