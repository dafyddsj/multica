// @vitest-environment node

import { describe, expect, it } from "vitest";
import { isAgentMailUsername, suggestedAgentMailUsername } from "./address";

describe("suggestedAgentMailUsername", () => {
  it("slugs an agent name into a legal local part", () => {
    expect(suggestedAgentMailUsername("Ada Mail")).toBe("ada-mail");
    expect(suggestedAgentMailUsername("  Support Bot  ")).toBe("support-bot");
  });

  it("returns empty when nothing usable remains", () => {
    expect(suggestedAgentMailUsername("???")).toBe("");
  });
});

describe("isAgentMailUsername", () => {
  it("accepts lowercase local parts", () => {
    expect(isAgentMailUsername("ada")).toBe(true);
    expect(isAgentMailUsername("ada.mail")).toBe(true);
  });

  it("rejects spaces and leading punctuation", () => {
    expect(isAgentMailUsername("Ada Mail")).toBe(false);
    expect(isAgentMailUsername("-ada")).toBe(false);
  });
});
