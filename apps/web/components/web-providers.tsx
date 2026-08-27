"use client";

import { useMemo, type ReactNode } from "react";
import { CoreProvider } from "@multica/core/platform";
import { createBrowserCookieLocaleAdapter } from "@multica/core/i18n/browser";
import type { LocaleResources, SupportedLocale } from "@multica/core/i18n";
import { useWelcomeStore } from "@multica/core/onboarding";
import { getApi } from "@multica/core/api";
import packageJson from "../package.json";
import { WebNavigationProvider } from "@/platform/navigation";
import { WebScrollRestorationProvider } from "@/platform/scroll-restoration";
import {
  setLoggedInCookie,
  clearLoggedInCookie,
} from "@/features/auth/auth-cookie";
import { detectWebOS } from "@/platform/client-os";
import { ClerkAuthGate } from "@/features/auth/clerk-shell";
import { useClerkSessionBridge } from "@/features/auth/clerk-session";

// Legacy token in localStorage → keep this session in token mode so users who
// logged in before the cookie-auth migration stay authed. They migrate to
// cookie mode on their next logout/login cycle (logout clears multica_token).
// Sunset: once telemetry shows <1% of sessions still carry multica_token,
// delete this branch and hard-code `cookieAuth` — the localStorage token is
// XSS-exposed and is the exact thing the cookie migration exists to remove.
function hasLegacyToken(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return Boolean(window.localStorage.getItem("multica_token"));
  } catch {
    return false;
  }
}

// Derive WebSocket URL from the page origin so self-hosted / LAN deployments
// work without an explicit runtime wsUrl. The Next.js runtime proxy handles
// /ws -> backend when the deployment keeps WebSockets same-origin.
function deriveWsUrl(): string | undefined {
  if (typeof window === "undefined") return undefined;
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/ws`;
}

// Build-time version preferred (CI sets NEXT_PUBLIC_APP_VERSION to a git tag
// or sha so different deploys are distinguishable in server logs); fall back
// to the package.json version so local dev still reports something useful.
const WEB_VERSION =
  process.env.NEXT_PUBLIC_APP_VERSION || packageJson.version || "dev";

function resetLocalSession() {
  // welcome-store holds the transient post-onboarding signal. Must
  // clear on logout so user B logging into the same browser doesn't
  // inherit user A's signal and have <WelcomeAfterOnboarding /> fire
  // listAgents / createIssue against a workspace user B doesn't even
  // belong to. The store's own docstring promises this reset; this
  // is where it gets wired.
  useWelcomeStore.getState().reset();
  clearLoggedInCookie();
  try {
    getApi().setToken(null);
    getApi().setTokenProvider(null);
  } catch {
    // CoreProvider has not created the client yet.
  }
}

export function WebProviders({
  children,
  locale,
  resources,
  apiBaseUrl,
  wsUrl,
}: {
  children: React.ReactNode;
  locale: SupportedLocale;
  resources: Record<string, LocaleResources>;
  apiBaseUrl?: string;
  wsUrl?: string;
}) {
  const cookieAuth = !hasLegacyToken();
  // Stable identity reference so downstream effects keyed on it don't see a
  // new object on every parent render.
  const identity = useMemo(
    () => ({ platform: "web", version: WEB_VERSION, os: detectWebOS() }),
    [],
  );
  const localeAdapter = useMemo(() => createBrowserCookieLocaleAdapter(), []);
  const resolvedWsUrl = wsUrl || deriveWsUrl();

  return (
    <ClerkAuthGate apiBaseUrl={apiBaseUrl}>
      {(clerkKey) =>
        clerkKey ? (
          <ClerkReadyProviders
            apiBaseUrl={apiBaseUrl}
            wsUrl={resolvedWsUrl}
            cookieAuth={cookieAuth}
            identity={identity}
            locale={locale}
            resources={resources}
            localeAdapter={localeAdapter}
          >
            {children}
          </ClerkReadyProviders>
        ) : (
          <NativeProviders
            apiBaseUrl={apiBaseUrl}
            wsUrl={resolvedWsUrl}
            cookieAuth={cookieAuth}
            identity={identity}
            locale={locale}
            resources={resources}
            localeAdapter={localeAdapter}
          >
            {children}
          </NativeProviders>
        )
      }
    </ClerkAuthGate>
  );
}

function NativeProviders({
  children,
  ...props
}: {
  children: ReactNode;
  apiBaseUrl?: string;
  wsUrl?: string;
  cookieAuth: boolean;
  identity: { platform: string; version: string; os: ReturnType<typeof detectWebOS> };
  locale: SupportedLocale;
  resources: Record<string, LocaleResources>;
  localeAdapter: ReturnType<typeof createBrowserCookieLocaleAdapter>;
}) {
  return (
    <CoreProvider
      {...props}
      onLogin={setLoggedInCookie}
      onLogout={resetLocalSession}
    >
      <WebNavigationProvider>
        <WebScrollRestorationProvider>{children}</WebScrollRestorationProvider>
      </WebNavigationProvider>
    </CoreProvider>
  );
}

function ClerkReadyProviders({
  children,
  ...props
}: {
  children: ReactNode;
  apiBaseUrl?: string;
  wsUrl?: string;
  cookieAuth: boolean;
  identity: { platform: string; version: string; os: ReturnType<typeof detectWebOS> };
  locale: SupportedLocale;
  resources: Record<string, LocaleResources>;
  localeAdapter: ReturnType<typeof createBrowserCookieLocaleAdapter>;
}) {
  const clerk = useClerkSessionBridge();
  if (!clerk.loaded) return null;
  return (
    <CoreProvider
      {...props}
      resolveAccessToken={clerk.resolveAccessToken}
      onLogin={setLoggedInCookie}
      onLogout={() => {
        resetLocalSession();
        void clerk.signOut();
      }}
    >
      <WebNavigationProvider>
        <WebScrollRestorationProvider>{children}</WebScrollRestorationProvider>
      </WebNavigationProvider>
    </CoreProvider>
  );
}
