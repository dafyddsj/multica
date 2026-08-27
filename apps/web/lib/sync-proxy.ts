import type { NextRequest, NextResponse } from "next/server";
import { proxy } from "../proxy";

/**
 * Overlay-off tests call proxy() with Clerk keys deleted, so the result is
 * always the sync NextResponse from appProxy. TypeScript still sees the
 * clerkMiddleware union (Response | Promise<...>). Narrow here instead of
 * forcing every caller to await a path that is sync at runtime.
 */
export function syncProxy(req: NextRequest): NextResponse {
  const result = proxy(req);
  if (result instanceof Promise) {
    throw new Error("expected a synchronous NextResponse when Clerk overlay is off");
  }
  if (!result || !("headers" in result)) {
    throw new Error("expected NextResponse from proxy");
  }
  return result as NextResponse;
}
