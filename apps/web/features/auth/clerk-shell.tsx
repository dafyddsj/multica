"use client";

import { useEffect, useState, type ReactNode } from "react";
import { ClerkProvider } from "@clerk/clerk-react";
import { configStore } from "@multica/core/config";

/**
 * Fetches /api/config so web can decide Clerk vs native before mounting
 * CoreProvider. Seeds the shared config store so AuthInitializer does not
 * flash a second auth mode.
 */
export function ClerkAuthGate({
  apiBaseUrl,
  children,
}: {
  apiBaseUrl?: string;
  children: (clerkPublishableKey: string) => ReactNode;
}) {
  const [clerkKey, setClerkKey] = useState<string | null>(null);

  useEffect(() => {
    const url = `${apiBaseUrl ?? ""}/api/config`;
    let cancelled = false;
    fetch(url, { credentials: "include" })
      .then((res) => (res.ok ? res.json() : {}))
      .then((cfg: Record<string, unknown>) => {
        if (cancelled) return;
        const key =
          typeof cfg.clerk_publishable_key === "string"
            ? cfg.clerk_publishable_key
            : "";
        configStore.getState().setAuthConfig({
          allowSignup: cfg.allow_signup !== false,
          googleClientId:
            typeof cfg.google_client_id === "string" ? cfg.google_client_id : "",
          clerkPublishableKey: key,
          workspaceCreationDisabled: cfg.workspace_creation_disabled === true,
          vcsIntegrationAvailable: cfg.vcs_integration_available === true,
        });
        setClerkKey(key);
      })
      .catch(() => {
        if (!cancelled) setClerkKey("");
      });
    return () => {
      cancelled = true;
    };
  }, [apiBaseUrl]);

  if (clerkKey === null) return null;

  if (!clerkKey) return <>{children("")}</>;

  return (
    <ClerkProvider publishableKey={clerkKey}>
      {children(clerkKey)}
    </ClerkProvider>
  );
}
