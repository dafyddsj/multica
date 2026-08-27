export type ClerkOverlayKeys = {
  secretKey: string;
  publishableKey: string;
};

export function clerkOverlayKeys(
  env: NodeJS.ProcessEnv = process.env,
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
  env: NodeJS.ProcessEnv = process.env,
): string {
  return clerkOverlayKeys(env)?.publishableKey ?? "";
}

export function applyClerkPublishableKeyAlias(
  env: NodeJS.ProcessEnv = process.env,
): void {
  const publishable = env.CLERK_PUBLISHABLE_KEY?.trim() ?? "";
  if (publishable && !env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY?.trim()) {
    env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY = publishable;
  }
}
