// @vitest-environment node
import { describe, expect, it } from "vitest";
import {
  EMPTY_LIST_INITIATIVES_RESPONSE,
  InitiativeDetailSchema,
  ListInitiativesResponseSchema,
} from "@multica/core/api/schemas";
import type { ListInitiativesResponse } from "@multica/core/types";
import { parseWithFallback } from "@/lib/parse-response";
import { EMPTY_INITIATIVE, PinnedItemSchema } from "./schemas";

describe("initiative list schema", () => {
  const ENDPOINT = { endpoint: "GET /api/initiatives" };

  it("keeps project_count, issue_count, and done_count from the server", () => {
    const parsed = parseWithFallback<ListInitiativesResponse>(
      {
        initiatives: [
          {
            id: "init-1",
            workspace_id: "ws-1",
            title: "Platform",
            description: null,
            icon: "🎯",
            status: "in_progress",
            priority: "high",
            lead_type: null,
            lead_id: null,
            created_at: "2026-08-01T00:00:00Z",
            updated_at: "2026-08-02T00:00:00Z",
            project_count: 3,
            issue_count: 12,
            done_count: 4,
          },
        ],
        total: 1,
      },
      ListInitiativesResponseSchema,
      EMPTY_LIST_INITIATIVES_RESPONSE,
      ENDPOINT,
    );
    expect(parsed).not.toBe(EMPTY_LIST_INITIATIVES_RESPONSE);
    expect(parsed.initiatives[0]?.id).toBe("init-1");
    expect(parsed.initiatives[0]?.project_count).toBe(3);
    expect(parsed.initiatives[0]?.issue_count).toBe(12);
    expect(parsed.initiatives[0]?.done_count).toBe(4);
  });

  it("defaults missing counts to 0 without dropping the row", () => {
    const parsed = parseWithFallback<ListInitiativesResponse>(
      {
        initiatives: [
          {
            id: "init-2",
            workspace_id: "ws-1",
            title: "Search",
            description: null,
            icon: null,
            status: "planned",
            priority: "none",
            lead_type: null,
            lead_id: null,
            created_at: "2026-08-01T00:00:00Z",
            updated_at: "2026-08-01T00:00:00Z",
          },
        ],
        total: 1,
      },
      ListInitiativesResponseSchema,
      EMPTY_LIST_INITIATIVES_RESPONSE,
      ENDPOINT,
    );
    expect(parsed.initiatives).toHaveLength(1);
    expect(parsed.initiatives[0]?.project_count).toBe(0);
    expect(parsed.initiatives[0]?.issue_count).toBe(0);
    expect(parsed.initiatives[0]?.done_count).toBe(0);
  });

  it("keeps an unknown status string so the UI default branch can render it", () => {
    const parsed = parseWithFallback(
      {
        id: "init-3",
        workspace_id: "ws-1",
        title: "Future",
        description: null,
        icon: null,
        status: "blocked",
        priority: "critical",
        lead_type: null,
        lead_id: null,
        created_at: "2026-08-01T00:00:00Z",
        updated_at: "2026-08-01T00:00:00Z",
        project_count: 0,
        issue_count: 0,
        done_count: 0,
      },
      InitiativeDetailSchema,
      EMPTY_INITIATIVE,
      { endpoint: "GET /api/initiatives/:id" },
    );
    expect(parsed).not.toBe(EMPTY_INITIATIVE);
    expect(parsed.status).toBe("blocked");
    expect(parsed.priority).toBe("critical");
  });
});

describe("pin item_type", () => {
  it("preserves initiative instead of coercing it to issue or project", () => {
    const parsed = PinnedItemSchema.parse({
      id: "pin-1",
      item_type: "initiative",
      item_id: "init-1",
    });
    expect(parsed.item_type).toBe("initiative");
  });

  it("preserves unknown types so the pins screen can ignore them", () => {
    const parsed = PinnedItemSchema.parse({
      id: "pin-2",
      item_type: "view",
      item_id: "view-1",
    });
    expect(parsed.item_type).toBe("view");
  });
});
