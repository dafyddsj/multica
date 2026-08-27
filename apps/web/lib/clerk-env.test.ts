// @vitest-environment node
import { describe, expect, it } from "vitest";
import {
  applyClerkPublishableKeyAlias,
  clerkOverlayKeys,
  clerkPublishableKeyFromEnv,
  type ClerkEnvBag,
} from "./clerk-env";

describe("clerkOverlayKeys", () => {
  it("returns null unless both secret and publishable keys are set", () => {
    expect(clerkOverlayKeys({})).toBeNull();
    expect(clerkOverlayKeys({ CLERK_SECRET_KEY: "sk_test_x" })).toBeNull();
    expect(clerkOverlayKeys({ CLERK_PUBLISHABLE_KEY: "pk_test_x" })).toBeNull();
    expect(
      clerkOverlayKeys({
        CLERK_SECRET_KEY: "  ",
        CLERK_PUBLISHABLE_KEY: "pk_test_x",
      }),
    ).toBeNull();
  });

  it("returns trimmed keys when both are set", () => {
    expect(
      clerkOverlayKeys({
        CLERK_SECRET_KEY: "  sk_test_x  ",
        CLERK_PUBLISHABLE_KEY: "  pk_test_x  ",
      }),
    ).toEqual({ secretKey: "sk_test_x", publishableKey: "pk_test_x" });
  });

  it("prefers NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY when both publishable vars exist", () => {
    expect(
      clerkOverlayKeys({
        CLERK_SECRET_KEY: "sk_test_x",
        CLERK_PUBLISHABLE_KEY: "pk_test_api",
        NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY: "pk_test_next",
      }),
    ).toEqual({ secretKey: "sk_test_x", publishableKey: "pk_test_next" });
  });
});

describe("clerkPublishableKeyFromEnv", () => {
  it("is empty when the overlay is off", () => {
    expect(clerkPublishableKeyFromEnv({})).toBe("");
  });

  it("returns the publishable key when the overlay is on", () => {
    expect(
      clerkPublishableKeyFromEnv({
        CLERK_SECRET_KEY: "sk_test_x",
        CLERK_PUBLISHABLE_KEY: "pk_test_x",
      }),
    ).toBe("pk_test_x");
  });
});

describe("applyClerkPublishableKeyAlias", () => {
  it("copies CLERK_PUBLISHABLE_KEY into NEXT_PUBLIC_ when that slot is empty", () => {
    const env: ClerkEnvBag = { CLERK_PUBLISHABLE_KEY: "pk_test_x" };
    applyClerkPublishableKeyAlias(env);
    expect(env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY).toBe("pk_test_x");
  });

  it("does not overwrite an existing NEXT_PUBLIC_ publishable key", () => {
    const env: ClerkEnvBag = {
      CLERK_PUBLISHABLE_KEY: "pk_test_api",
      NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY: "pk_test_next",
    };
    applyClerkPublishableKeyAlias(env);
    expect(env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY).toBe("pk_test_next");
  });
});
