"use client";

import { useEffect } from "react";
import { useAuth, useClerk } from "@clerk/clerk-react";
import { getApi } from "@multica/core/api";

export function useClerkSessionBridge(): {
  loaded: boolean;
  resolveAccessToken: () => Promise<string | null>;
  signOut: () => Promise<void>;
} {
  const { isLoaded, isSignedIn, getToken } = useAuth();
  const { signOut } = useClerk();

  useEffect(() => {
    if (!isLoaded) return;
    try {
      const api = getApi();
      api.setTokenProvider(async () => {
        if (!isSignedIn) return null;
        return getToken();
      });
      return () => {
        api.setTokenProvider(null);
      };
    } catch {
      return undefined;
    }
  }, [isLoaded, isSignedIn, getToken]);

  return {
    loaded: isLoaded,
    resolveAccessToken: async () => {
      if (!isSignedIn) return null;
      return getToken();
    },
    signOut: async () => {
      await signOut();
    },
  };
}
