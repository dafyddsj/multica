"use client";

import { useEffect, useRef, useState } from "react";
import { SignIn, useAuth } from "@clerk/nextjs";
import { useAuthStore } from "@multica/core/auth";
import { api } from "@multica/core/api";
import { workspaceKeys } from "@multica/core/workspace/queries";
import { useQueryClient } from "@tanstack/react-query";
import { redirectToCliCallback } from "@multica/views/auth";
import { Card, CardDescription, CardHeader, CardTitle } from "@multica/ui/components/ui/card";
import { Loader2 } from "lucide-react";
import { setLoggedInCookie } from "@/features/auth/auth-cookie";
import { useT } from "@multica/views/i18n";

export function ClerkLogin({
  onSuccess,
  cliCallback,
}: {
  onSuccess: () => void | Promise<void>;
  cliCallback?: { url: string; state: string };
}) {
  const { isLoaded, isSignedIn, getToken } = useAuth();
  const qc = useQueryClient();
  const { t } = useT("auth");
  const [error, setError] = useState("");
  const completingRef = useRef(false);

  useEffect(() => {
    if (!isLoaded || !isSignedIn || completingRef.current) return;
    completingRef.current = true;

    void (async () => {
      try {
        const token = await getToken();
        if (!token) {
          throw new Error("Clerk session token was empty");
        }
        api.setToken(token);
        const user = await api.getMe();
        useAuthStore.setState({
          user,
          isLoading: false,
          status: "authenticated",
        });
        setLoggedInCookie();

        if (cliCallback) {
          const { token: cliToken } = await api.issueCliToken();
          redirectToCliCallback(cliCallback.url, cliToken, cliCallback.state);
          return;
        }

        try {
          const list = await api.listWorkspaces();
          qc.setQueryData(workspaceKeys.list(), list);
        } catch {
          // handleSuccess falls back to an empty list
        }
        await onSuccess();
      } catch (err) {
        completingRef.current = false;
        setError(err instanceof Error ? err.message : t(($) => $.web.desktop_handoff.prepare_failed));
      }
    })();
  }, [isLoaded, isSignedIn, getToken, cliCallback, onSuccess, qc, t]);

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Card className="w-full max-w-sm">
          <CardHeader className="text-center">
            <CardTitle className="text-display-sm">
              {t(($) => $.web.desktop_handoff.failed_title)}
            </CardTitle>
            <CardDescription>{error}</CardDescription>
          </CardHeader>
        </Card>
      </div>
    );
  }

  if (!isLoaded || isSignedIn) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      {/* Hash routing keeps the widget on /login. Pin Clerk's post-auth
          redirects too: dashboard "home" is `/`, and without these props
          the hosted widget navigates there after OTP before handleSuccess
          can run. /login owns destination (onboarding vs workspace). */}
      <SignIn
        routing="hash"
        forceRedirectUrl="/login"
        fallbackRedirectUrl="/login"
        signUpForceRedirectUrl="/login"
        signUpFallbackRedirectUrl="/login"
      />
    </div>
  );
}
