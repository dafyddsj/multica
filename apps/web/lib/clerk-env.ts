export type ClerkOverlayKeys = {
  secretKey: string;
  publishableKey: string;
};

/** Env bag for overlay gating. process.env is assignable; tests can pass partials. */
export type ClerkEnvBag = Record<string, string | undefined>;

export function clerkOverlayKeys(
  env: ClerkEnvBag = process.env,
): ClerkOverlayKeys | null {
  const secretKey = env.CLERK_SECRET_KEY?.trim() ?? "";
  const publishableKey = (
    env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY ??
    env.CLERK_PUBLISHABLE_KEY ??
    ""
  ).trim();
  if (!secretKey || !publishableKey) return null;
  return { secretKey, publishableKey };
}

export function clerkPublishableKeyFromEnv(
  env: ClerkEnvBag = process.env,
): string {
  return clerkOverlayKeys(env)?.publishableKey ?? "";
}

export function applyClerkPublishableKeyAlias(
  env: ClerkEnvBag = process.env,
): void {
  const publishable = env.CLERK_PUBLISHABLE_KEY?.trim() ?? "";
  if (publishable && !env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY?.trim()) {
    env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY = publishable;
  }
}
