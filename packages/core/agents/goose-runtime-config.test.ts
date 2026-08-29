// @vitest-environment node

import { describe, expect, it } from "vitest";
import {
  applyGooseProvider,
  parseGooseRuntimeConfig,
} from "./goose-runtime-config";

describe("parseGooseRuntimeConfig", () => {
  it("returns empty for null, arrays, and non-objects", () => {
    expect(parseGooseRuntimeConfig(null)).toEqual({});
    expect(parseGooseRuntimeConfig(undefined)).toEqual({});
    expect(parseGooseRuntimeConfig([])).toEqual({});
    expect(parseGooseRuntimeConfig("ollama")).toEqual({});
    expect(parseGooseRuntimeConfig(1)).toEqual({});
  });

  it("reads goose_provider, trims, and omits empty", () => {
    expect(parseGooseRuntimeConfig({})).toEqual({});
    expect(parseGooseRuntimeConfig({ goose_provider: " ollama " })).toEqual({
      provider: "ollama",
    });
    expect(parseGooseRuntimeConfig({ goose_provider: "   " })).toEqual({});
    expect(parseGooseRuntimeConfig({ goose_provider: "" })).toEqual({});
  });

  it("drops unknown keys and ignores a bare provider key", () => {
    expect(
      parseGooseRuntimeConfig({
        goose_provider: "openrouter",
        mode: "local",
        provider: "ollama",
      }),
    ).toEqual({ provider: "openrouter" });
  });

  it("never throws on malformed values", () => {
    expect(parseGooseRuntimeConfig({ goose_provider: 12 })).toEqual({});
    expect(parseGooseRuntimeConfig({ goose_provider: { id: "ollama" } })).toEqual(
      {},
    );
  });
});

describe("applyGooseProvider", () => {
  it("sets goose_provider and preserves sibling keys", () => {
    expect(applyGooseProvider({ mode: "local" }, " ollama ")).toEqual({
      mode: "local",
      goose_provider: "ollama",
    });
  });

  it("deletes goose_provider when the value is empty after trim", () => {
    expect(
      applyGooseProvider({ mode: "local", goose_provider: "ollama" }, "  "),
    ).toEqual({ mode: "local" });
    expect(applyGooseProvider(undefined, "")).toEqual({});
  });

  it("does not treat a sibling provider key as the Goose field", () => {
    expect(applyGooseProvider({ provider: "keep-me" }, "openrouter")).toEqual({
      provider: "keep-me",
      goose_provider: "openrouter",
    });
  });
});
