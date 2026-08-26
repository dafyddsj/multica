import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "@multica/core/i18n/react";
import enInitiatives from "../../locales/en/initiatives.json";
import enIssues from "../../locales/en/issues.json";
import { InitiativePicker } from "./initiative-picker";
import { PillButton } from "../../common/pill-button";

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: [
      { id: "initiative-1", title: "Launch Command Center", icon: null },
      { id: "initiative-2", title: "Mobile Web", icon: null },
    ],
  }),
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@multica/core/initiatives/queries", () => ({
  initiativeListOptions: () => ({ queryKey: ["initiatives"] }),
}));

vi.mock("./initiative-icon", () => ({
  InitiativeIcon: () => <span data-testid="initiative-icon" />,
}));

function withI18n(children: React.ReactNode) {
  return (
    <I18nProvider locale="en" resources={{ en: { initiatives: enInitiatives, issues: enIssues } }}>
      {children}
    </I18nProvider>
  );
}

function renderPicker(props: Partial<React.ComponentProps<typeof InitiativePicker>> = {}) {
  return render(
    withI18n(
      <InitiativePicker
        initiativeId="initiative-1"
        onUpdate={props.onUpdate ?? vi.fn()}
        triggerRender={<PillButton />}
        {...props}
      />,
    ),
  );
}

const SEARCH_PLACEHOLDER = "Search initiatives...";

describe("InitiativePicker", () => {
  it("renders the trigger alone — the pill carries no inline clear control", () => {
    renderPicker();

    expect(screen.getAllByRole("button")).toHaveLength(1);
    expect(screen.getByRole("button", { name: /launch command center/i })).toBeInTheDocument();
  });

  it("clears the selection from the No initiative row", async () => {
    const user = userEvent.setup();
    const onUpdate = vi.fn();

    renderPicker({ onUpdate });

    await user.click(screen.getByRole("button", { name: /launch command center/i }));
    await user.click(await screen.findByRole("button", { name: /no initiative/i }));

    expect(onUpdate).toHaveBeenCalledWith({ initiative_id: null });
  });

  it("keeps the empty row first while searching", async () => {
    const user = userEvent.setup();
    const onUpdate = vi.fn();

    renderPicker({ onUpdate });

    await user.click(screen.getByRole("button", { name: /launch command center/i }));
    await user.type(await screen.findByPlaceholderText(SEARCH_PLACEHOLDER), "zzzznomatch");

    await user.click(screen.getByRole("button", { name: /no initiative/i }));
    expect(onUpdate).toHaveBeenCalledWith({ initiative_id: null });
  });

  it("keeps the empty row ahead of the filtered matches", async () => {
    const user = userEvent.setup();

    renderPicker({ initiativeId: null });

    await user.click(screen.getByRole("button", { name: /no initiative/i }));
    expect(screen.getAllByRole("button", { name: /no initiative/i })).toHaveLength(2);

    await user.type(await screen.findByPlaceholderText(SEARCH_PLACEHOLDER), "mobile");
    expect(screen.getAllByRole("button", { name: /no initiative/i })).toHaveLength(2);
    expect(screen.getByRole("button", { name: /mobile web/i })).toBeInTheDocument();
  });

  it("locks every mutation path when disabled", async () => {
    const user = userEvent.setup();
    const onUpdate = vi.fn();

    const { unmount } = render(
      withI18n(<InitiativePicker initiativeId="initiative-1" onUpdate={onUpdate} disabled />),
    );

    const trigger = screen.getByRole("button", { name: /launch command center/i });
    expect(trigger).toBeDisabled();

    await user.click(trigger);
    expect(screen.queryByPlaceholderText(SEARCH_PLACEHOLDER)).not.toBeInTheDocument();
    expect(onUpdate).not.toHaveBeenCalled();

    unmount();
    render(withI18n(<InitiativePicker initiativeId="initiative-1" onUpdate={onUpdate} disabled open />));
    expect(screen.queryByPlaceholderText(SEARCH_PLACEHOLDER)).not.toBeInTheDocument();
    expect(onUpdate).not.toHaveBeenCalled();
  });
});
