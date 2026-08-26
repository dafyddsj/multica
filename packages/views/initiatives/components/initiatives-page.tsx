"use client";

import { useCallback, useMemo, useState, type MouseEvent } from "react";
import {
  ArrowDown,
  ArrowUp,
  ChevronDown,
  ExternalLink,
  Filter,
  Target,
  LayoutGrid,
  MoreHorizontal,
  Pin,
  PinOff,
  Plus,
  Rows3,
  Search,
  Trash2,
  X,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  initiativeListOptions,
  useUpdateInitiative,
  useDeleteInitiative,
  useInitiativeViewStore,
  type InitiativeColumnKey,
  type InitiativeListFilters,
  type InitiativeSortField,
  type InitiativeViewMode,
} from "@multica/core/initiatives";
import {
  pinListOptions,
  useCreatePin,
  useDeletePin,
} from "@multica/core/pins";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { useAuthStore } from "@multica/core/auth";
import { useActorName } from "@multica/core/workspace/hooks";
import { memberListOptions } from "@multica/core/workspace/queries";
import { useModalStore } from "@multica/core/modals";
import { AppLink, useIntentNavigate, useRowLink } from "../../navigation";
import { ActorAvatar } from "../../common/actor-avatar";
import { FILTER_ITEM_CLASS, HoverCheck } from "../../common/hover-check";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { Button } from "@multica/ui/components/ui/button";
import { Checkbox } from "@multica/ui/components/ui/checkbox";
import { Input } from "@multica/ui/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import {
  ListGrid,
  ListGridCell,
  ListGridHeader,
  ListGridHeaderCell,
  ListGridRow,
  LIST_GRID_BOTTOM_CLEARANCE,
  type ListGridSortDirection,
} from "@multica/ui/components/ui/list-grid";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@multica/ui/components/ui/popover";
import { Switch } from "@multica/ui/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@multica/ui/components/ui/tooltip";
import type {
  MemberWithUser,
  Initiative,
  InitiativePriority,
  InitiativeStatus,
  UpdateInitiativeRequest,
} from "@multica/core/types";
import {
  CollectionPageHeader,
  CollectionPageHeaderAction,
  CollectionPageState,
} from "../../layout/collection-page";
import { InitiativeIcon } from "./initiative-icon";
import { useT } from "../../i18n";
import { matchesPinyin } from "../../editor/extensions/pinyin-match";
import { useFormatRelativeDate } from "./labels";
import { InitiativeStatusBadge, InitiativePriorityBadge } from "./initiative-badge";
import { InitiativeLeadPicker } from "./initiative-lead-picker";
import { PAGE_GUTTER, PAGE_TOOLBAR } from "../../layout/page-header";
import { cn } from "@multica/ui/lib/utils";

const PRIORITY_ORDER: Record<InitiativePriority, number> = {
  urgent: 4,
  high: 3,
  medium: 2,
  low: 1,
  none: 0,
};
const STATUS_ORDER: Record<InitiativeStatus, number> = {
  planned: 0,
  in_progress: 1,
  paused: 2,
  completed: 3,
  cancelled: 4,
};

const progressOf = (p: Initiative) =>
  p.issue_count > 0 ? p.done_count / p.issue_count : -1;

function leadFilterValue(p: Initiative): string | null {
  return p.lead_type && p.lead_id ? `${p.lead_type}:${p.lead_id}` : null;
}

const COLUMN_WIDTHS: Record<InitiativeColumnKey, number> = {
  priority: 116,
  progress: 88,
  lead: 132,
  projects: 80,
  created: 104,
};

const FIXED_TRACKS_WIDTH = 384 + 10 * 12;

// MUST be a literal string — Tailwind can't see interpolated `grid-cols-[...]`
// arbitrary values, so an interpolated width silently drops the whole template
// and the grid collapses to one column.
const GRID_COLS =
  "grid-cols-[0.75rem_1rem_minmax(120px,1fr)_116px_1.75rem_0.75rem] " +
  "@2xl:grid-cols-[0.75rem_1rem_minmax(200px,1fr)_116px_var(--inc-priority)_var(--inc-progress)_var(--inc-lead)_var(--inc-projects)_var(--inc-created)_1.75rem_0.75rem]";

const stopRowNavigation = (e: MouseEvent) => e.stopPropagation();

function columnTrackVars(
  isVisible: (key: InitiativeColumnKey) => boolean,
): React.CSSProperties {
  const width = (key: InitiativeColumnKey) =>
    isVisible(key) ? `${COLUMN_WIDTHS[key]}px` : "0px";
  const minWidth =
    FIXED_TRACKS_WIDTH +
    (Object.keys(COLUMN_WIDTHS) as InitiativeColumnKey[]).reduce(
      (sum, key) => sum + (isVisible(key) ? COLUMN_WIDTHS[key] : 0),
      0,
    );
  return {
    "--inc-priority": width("priority"),
    "--inc-progress": width("progress"),
    "--inc-lead": width("lead"),
    "--inc-projects": width("projects"),
    "--inc-created": width("created"),
    "--inc-minw": `${minWidth}px`,
  } as React.CSSProperties;
}

function ProgressRing({ initiative }: { initiative: Initiative }) {
  if (initiative.issue_count === 0) {
    return <span className="text-caption text-faint-foreground">—</span>;
  }
  const pct = Math.round((initiative.done_count / initiative.issue_count) * 100);
  return (
    <span className="flex items-center gap-1.5">
      <span className="relative h-3.5 w-3.5">
        <svg className="h-3.5 w-3.5 -rotate-90" viewBox="0 0 16 16">
          <circle className="text-muted" strokeWidth="2" stroke="currentColor" fill="none" r="6" cx="8" cy="8" />
          <circle
            className="text-emerald-500"
            strokeWidth="2"
            stroke="currentColor"
            fill="none"
            r="6"
            cx="8"
            cy="8"
            strokeDasharray={`${pct * 0.377} 37.7`}
            strokeLinecap="round"
          />
        </svg>
      </span>
      <span className="text-caption tabular-nums text-muted-foreground">
        {initiative.done_count}/{initiative.issue_count}
      </span>
    </span>
  );
}

function InitiativeRowActions({
  initiative,
  pinned,
  canDelete,
}: {
  initiative: Initiative;
  pinned: boolean;
  canDelete: boolean;
}) {
  const { t } = useT("initiatives");
  const { t: tCommon } = useT("common");
  const wsPaths = useWorkspacePaths();
  const intentNavigate = useIntentNavigate();
  const createPin = useCreatePin();
  const deletePin = useDeletePin();
  const deleteInitiative = useDeleteInitiative();
  const [deleteOpen, setDeleteOpen] = useState(false);

  const togglePin = () => {
    if (pinned) deletePin.mutate({ itemType: "initiative", itemId: initiative.id });
    else createPin.mutate({ item_type: "initiative", item_id: initiative.id });
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <button
              type="button"
              aria-label={t(($) => $.page.row_menu)}
              className="flex size-7 items-center justify-center rounded-md text-muted-foreground opacity-0 transition-opacity hover:bg-accent hover:text-accent-foreground group-hover/row:opacity-100 data-popup-open:bg-accent data-popup-open:opacity-100 data-popup-open:text-accent-foreground"
            >
              <MoreHorizontal className="size-4" />
            </button>
          }
        />
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem
            onClick={() =>
              intentNavigate(
                wsPaths.initiativeDetail(initiative.id),
                "foreground-tab",
                initiative.title,
              )
            }
          >
            <ExternalLink className="size-3.5" />
            {tCommon(($) => $.navigation.open_in_new_tab)}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={togglePin}>
            {pinned ? (
              <PinOff className="size-3.5" />
            ) : (
              <Pin className="size-3.5" />
            )}
            {pinned ? t(($) => $.page.unpin) : t(($) => $.page.pin)}
          </DropdownMenuItem>
          {canDelete && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                variant="destructive"
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 className="size-3.5" />
                {t(($) => $.page.delete)}
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t(($) => $.delete_dialog.title)}</DialogTitle>
            <DialogDescription>
              {t(($) => $.delete_dialog.description)}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setDeleteOpen(false)}
            >
              {t(($) => $.delete_dialog.cancel)}
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => {
                deleteInitiative.mutate(initiative.id, {
                  onError: (err) =>
                    toast.error(
                      err instanceof Error ? err.message : String(err),
                    ),
                });
                setDeleteOpen(false);
              }}
            >
              {t(($) => $.delete_dialog.confirm)}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function CheckboxCell({
  checked,
  onToggle,
}: {
  checked: boolean;
  onToggle: () => void;
}) {
  return (
    <ListGridCell className="justify-center px-0">
      <button
        type="button"
        aria-pressed={checked}
        onClick={(e) => {
          stopRowNavigation(e);
          onToggle();
        }}
        onAuxClick={stopRowNavigation}
        className={`-m-1.5 flex items-center p-1.5 ${
          checked ? "" : "opacity-0 transition-opacity group-hover/row:opacity-100"
        }`}
      >
        <Checkbox checked={checked} tabIndex={-1} className="pointer-events-none" />
      </button>
    </ListGridCell>
  );
}

function InitiativeTableRow({
  initiative,
  pinned,
  canDelete,
  isColVisible,
  selected,
  onToggleSelect,
  rowHref,
  rowLink,
}: {
  initiative: Initiative;
  pinned: boolean;
  canDelete: boolean;
  isColVisible: (key: InitiativeColumnKey) => boolean;
  selected: boolean;
  onToggleSelect: () => void;
  rowHref: string;
  rowLink: ReturnType<typeof useRowLink>;
}) {
  const formatRelativeDate = useFormatRelativeDate();
  const updateInitiative = useUpdateInitiative();
  const handleUpdate = useCallback(
    (data: UpdateInitiativeRequest) => updateInitiative.mutate({ id: initiative.id, ...data }),
    [initiative.id, updateInitiative],
  );

  return (
    <ListGridRow
      className={`h-11 cursor-pointer ${selected ? "bg-accent/30" : ""}`}
      {...rowLink(rowHref, initiative.title)}
    >
      <CheckboxCell checked={selected} onToggle={onToggleSelect} />
      <ListGridCell className="gap-2">
        <InitiativeIcon initiative={initiative} size="sm" />
        <span className="min-w-0 truncate text-body font-medium">
          {initiative.title}
        </span>
      </ListGridCell>

      <ListGridCell onClick={stopRowNavigation} onAuxClick={stopRowNavigation}>
        <InitiativeStatusBadge initiative={initiative} handleUpdate={handleUpdate} align="start" />
      </ListGridCell>

      {isColVisible("priority") ? (
        <ListGridCell className="hidden @2xl:flex" onClick={stopRowNavigation} onAuxClick={stopRowNavigation}>
          <InitiativePriorityBadge initiative={initiative} handleUpdate={handleUpdate} align="start" />
        </ListGridCell>
      ) : (
        <ListGridCell className="hidden px-0 @2xl:flex" />
      )}

      {isColVisible("progress") ? (
        <ListGridCell className="hidden @2xl:flex">
          <ProgressRing initiative={initiative} />
        </ListGridCell>
      ) : (
        <ListGridCell className="hidden px-0 @2xl:flex" />
      )}

      {isColVisible("lead") ? (
        <ListGridCell className="hidden @2xl:flex" onClick={stopRowNavigation} onAuxClick={stopRowNavigation}>
          <InitiativeLeadPicker
            initiative={initiative}
            handleUpdate={handleUpdate}
            align="start"
            renderTrigger={(leadName) => (
              <button
                type="button"
                className="flex min-w-0 items-center gap-1.5 rounded px-1 py-0.5 transition-colors hover:bg-accent/60"
              >
                {initiative.lead_type && initiative.lead_id ? (
                  <ActorAvatar actorType={initiative.lead_type} actorId={initiative.lead_id} size="sm" enableHoverCard />
                ) : (
                  <span className="inline-flex h-[18px] w-[18px] rounded-full border border-dashed border-muted-foreground/30" />
                )}
                <span className="min-w-0 truncate text-caption text-muted-foreground">
                  {leadName ?? "—"}
                </span>
              </button>
            )}
          />
        </ListGridCell>
      ) : (
        <ListGridCell className="hidden px-0 @2xl:flex" />
      )}

      {isColVisible("projects") ? (
        <ListGridCell className="hidden justify-end font-mono text-caption tabular-nums text-muted-foreground @2xl:flex">
          {initiative.project_count}
        </ListGridCell>
      ) : (
        <ListGridCell className="hidden px-0 @2xl:flex" />
      )}

      {isColVisible("created") ? (
        <ListGridCell className="hidden whitespace-nowrap text-caption tabular-nums text-muted-foreground @2xl:flex">
          {formatRelativeDate(initiative.created_at)}
        </ListGridCell>
      ) : (
        <ListGridCell className="hidden px-0 @2xl:flex" />
      )}

      <ListGridCell className="justify-end px-0">
        <span onClick={stopRowNavigation} onAuxClick={stopRowNavigation} className="flex items-center">
          <InitiativeRowActions initiative={initiative} pinned={pinned} canDelete={canDelete} />
        </span>
      </ListGridCell>
    </ListGridRow>
  );
}

function InitiativeTableHeader({
  sortField,
  sortDirection,
  onSort,
  isColVisible,
  allSelected,
  someSelected,
  onToggleAll,
}: {
  sortField: InitiativeSortField;
  sortDirection: ListGridSortDirection;
  onSort: (field: InitiativeSortField) => void;
  isColVisible: (key: InitiativeColumnKey) => boolean;
  allSelected: boolean;
  someSelected: boolean;
  onToggleAll: () => void;
}) {
  const { t } = useT("initiatives");
  const sorted = (field: InitiativeSortField) =>
    sortField === field ? sortDirection : false;
  const anySelected = allSelected || someSelected;
  return (
    <ListGridHeader>
      <div className="flex items-center justify-center">
        <button
          type="button"
          aria-pressed={allSelected}
          onClick={onToggleAll}
          className={`-m-1.5 flex items-center p-1.5 ${
            anySelected ? "" : "opacity-0 transition-opacity group-hover/header:opacity-100"
          }`}
        >
          <Checkbox
            checked={allSelected}
            indeterminate={someSelected && !allSelected}
            tabIndex={-1}
            className="pointer-events-none"
          />
        </button>
      </div>
      <ListGridHeaderCell sorted={sorted("name")} onSort={() => onSort("name")}>
        {t(($) => $.table.name)}
      </ListGridHeaderCell>
      <ListGridHeaderCell sorted={sorted("status")} onSort={() => onSort("status")}>
        {t(($) => $.table.status)}
      </ListGridHeaderCell>
      {isColVisible("priority") ? (
        <ListGridHeaderCell
          className="hidden @2xl:flex"
          sorted={sorted("priority")}
          onSort={() => onSort("priority")}
        >
          {t(($) => $.table.priority)}
        </ListGridHeaderCell>
      ) : (
        <ListGridHeaderCell className="hidden px-0 @2xl:flex" />
      )}
      {isColVisible("progress") ? (
        <ListGridHeaderCell
          className="hidden @2xl:flex"
          sorted={sorted("progress")}
          onSort={() => onSort("progress")}
        >
          {t(($) => $.table.progress)}
        </ListGridHeaderCell>
      ) : (
        <ListGridHeaderCell className="hidden px-0 @2xl:flex" />
      )}
      {isColVisible("lead") ? (
        <ListGridHeaderCell className="hidden @2xl:flex">
          {t(($) => $.table.lead)}
        </ListGridHeaderCell>
      ) : (
        <ListGridHeaderCell className="hidden px-0 @2xl:flex" />
      )}
      {isColVisible("projects") ? (
        <ListGridHeaderCell className="hidden justify-end @2xl:flex" align="right">
          {t(($) => $.table.projects)}
        </ListGridHeaderCell>
      ) : (
        <ListGridHeaderCell className="hidden px-0 @2xl:flex" />
      )}
      {isColVisible("created") ? (
        <ListGridHeaderCell
          className="hidden @2xl:flex"
          sorted={sorted("created")}
          onSort={() => onSort("created")}
        >
          {t(($) => $.table.created)}
        </ListGridHeaderCell>
      ) : (
        <ListGridHeaderCell className="hidden px-0 @2xl:flex" />
      )}
      <span aria-hidden="true" />
    </ListGridHeader>
  );
}

function InitiativeCard({
  initiative,
  pinned,
  canDelete,
}: {
  initiative: Initiative;
  pinned: boolean;
  canDelete: boolean;
}) {
  const { t } = useT("initiatives");
  const wsPaths = useWorkspacePaths();
  const formatRelativeDate = useFormatRelativeDate();
  const updateInitiative = useUpdateInitiative();
  const handleUpdate = useCallback(
    (data: UpdateInitiativeRequest) => updateInitiative.mutate({ id: initiative.id, ...data }),
    [initiative.id, updateInitiative],
  );
  const progressPercent =
    initiative.issue_count > 0
      ? Math.round((initiative.done_count / initiative.issue_count) * 100)
      : 0;

  return (
    <div className="group/card group/row flex flex-col rounded-md border bg-card transition-colors hover:border-primary/50">
      <div className="p-3 pb-2">
        <div className="flex items-center gap-2">
          <AppLink
            href={wsPaths.initiativeDetail(initiative.id)}
            className="flex min-w-0 flex-1 items-center gap-2"
          >
            <InitiativeIcon initiative={initiative} size="sm" />
            <h3 className="truncate text-body font-medium">{initiative.title}</h3>
          </AppLink>
          <InitiativeRowActions initiative={initiative} pinned={pinned} canDelete={canDelete} />
          <InitiativeStatusBadge initiative={initiative} handleUpdate={handleUpdate} triggerClassName="shrink-0" />
        </div>

        {initiative.issue_count > 0 ? (
          <div className="flex items-center justify-end gap-1.5 pt-2">
            <div className="relative h-4 w-4">
              <svg className="h-4 w-4 -rotate-90" viewBox="0 0 16 16">
                <circle className="text-muted" strokeWidth="2" stroke="currentColor" fill="none" r="6" cx="8" cy="8" />
                <circle
                  className="text-emerald-500"
                  strokeWidth="2"
                  stroke="currentColor"
                  fill="none"
                  r="6"
                  cx="8"
                  cy="8"
                  strokeDasharray={`${progressPercent * 0.377} 37.7`}
                  strokeLinecap="round"
                />
              </svg>
            </div>
            <span className="text-micro tabular-nums text-muted-foreground">
              {initiative.done_count}/{initiative.issue_count}
            </span>
          </div>
        ) : (
          <span className="flex justify-end pt-2 text-micro text-muted-foreground">
            {t(($) => $.detail.no_issues_yet)}
          </span>
        )}
      </div>

      <div className="mt-0 flex items-center justify-between border-t px-3 pb-3 pt-2">
        <InitiativeLeadPicker
          initiative={initiative}
          handleUpdate={handleUpdate}
          renderTrigger={(leadName) => (
            <button type="button" className="-mx-1.5 flex items-center gap-1.5 rounded px-1.5 py-0.5 transition-colors hover:bg-accent/60">
              {initiative.lead_type && initiative.lead_id ? (
                <ActorAvatar actorType={initiative.lead_type} actorId={initiative.lead_id} size="sm" enableHoverCard />
              ) : (
                <span className="inline-flex h-5 w-5 rounded-full border border-dashed border-muted-foreground/30" />
              )}
              <span className="max-w-[60px] truncate text-micro text-muted-foreground">
                {leadName ?? t(($) => $.lead.no_lead)}
              </span>
            </button>
          )}
        />
        <div className="flex items-center gap-2">
          <InitiativePriorityBadge initiative={initiative} handleUpdate={handleUpdate} align="start" />
          <span className="text-micro text-muted-foreground">
            {formatRelativeDate(initiative.created_at)}
          </span>
        </div>
      </div>
    </div>
  );
}

const STATUS_VALUES: InitiativeStatus[] = [
  "planned",
  "in_progress",
  "paused",
  "completed",
  "cancelled",
];
const PRIORITY_VALUES: InitiativePriority[] = ["urgent", "high", "medium", "low", "none"];
const COLUMN_KEYS: InitiativeColumnKey[] = ["priority", "progress", "lead", "projects", "created"];
const SORT_FIELDS: InitiativeSortField[] = ["name", "priority", "status", "progress", "created"];

function countActiveFilters(f: InitiativeListFilters): number {
  let c = 0;
  if (f.statuses.length) c++;
  if (f.priorities.length) c++;
  if (f.leads.length) c++;
  return c;
}

function InitiativeBatchToolbar({
  rows,
  pinnedIds,
  canDelete,
  onClear,
}: {
  rows: Initiative[];
  pinnedIds: Set<string>;
  canDelete: boolean;
  onClear: () => void;
}) {
  const { t } = useT("initiatives");
  const createPin = useCreatePin();
  const deleteInitiative = useDeleteInitiative();
  const [confirmDelete, setConfirmDelete] = useState(false);

  if (rows.length === 0) return null;
  const anyUnpinned = rows.some((p) => !pinnedIds.has(p.id));

  return (
    <>
      <div className="absolute bottom-6 left-1/2 z-50 flex -translate-x-1/2 items-center gap-1 rounded-lg border bg-background px-2 py-1.5 shadow-lg max-md:above-chat-launcher">
        <div className="mr-1 flex items-center gap-1.5 border-r pl-1 pr-2">
          <span className="text-body font-medium">
            {t(($) => $.page.selected, { count: rows.length })}
          </span>
          <button
            type="button"
            aria-label={t(($) => $.page.clear_selection)}
            onClick={onClear}
            className="rounded p-0.5 transition-colors hover:bg-accent"
          >
            <X className="size-3.5 text-muted-foreground" />
          </button>
        </div>
        {anyUnpinned && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              for (const p of rows) {
                if (!pinnedIds.has(p.id)) {
                  createPin.mutate({ item_type: "initiative", item_id: p.id });
                }
              }
              onClear();
            }}
          >
            <Pin className="mr-1 size-3.5" />
            {t(($) => $.page.pin)}
          </Button>
        )}
        {canDelete && (
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={() => setConfirmDelete(true)}
          >
            <Trash2 className="mr-1 size-3.5" />
            {t(($) => $.page.delete)}
          </Button>
        )}
      </div>

      <Dialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t(($) => $.delete_dialog.title)}</DialogTitle>
            <DialogDescription>{t(($) => $.delete_dialog.description)}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" size="sm" onClick={() => setConfirmDelete(false)}>
              {t(($) => $.delete_dialog.cancel)}
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => {
                for (const p of rows) deleteInitiative.mutate(p.id);
                setConfirmDelete(false);
                onClear();
              }}
            >
              {t(($) => $.delete_dialog.confirm)}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function InitiativesPage() {
  const { t } = useT("initiatives");
  const wsId = useWorkspaceId();
  const wsPaths = useWorkspacePaths();
  const rowLink = useRowLink();
  const currentUser = useAuthStore((s) => s.user);
  const { getActorName } = useActorName();

  const viewMode = useInitiativeViewStore((s) => s.viewMode);
  const setViewMode = useInitiativeViewStore((s) => s.setViewMode);
  const sortField = useInitiativeViewStore((s) => s.sortField);
  const sortDirection = useInitiativeViewStore((s) => s.sortDirection);
  const hiddenColumns = useInitiativeViewStore((s) => s.hiddenColumns);
  const filters = useInitiativeViewStore((s) => s.filters);
  const toggleSort = useInitiativeViewStore((s) => s.toggleSort);
  const setSortField = useInitiativeViewStore((s) => s.setSortField);
  const setSortDirection = useInitiativeViewStore((s) => s.setSortDirection);
  const toggleColumn = useInitiativeViewStore((s) => s.toggleColumn);
  const toggleFilter = useInitiativeViewStore((s) => s.toggleFilter);
  const clearFilters = useInitiativeViewStore((s) => s.clearFilters);
  const isCompact = viewMode === "compact";
  const isColVisible = (key: InitiativeColumnKey) => !hiddenColumns.includes(key);

  const { data: initiatives = [], isLoading } = useQuery(initiativeListOptions(wsId));
  const { data: members = [] } = useQuery(memberListOptions(wsId));
  const { data: pins = [] } = useQuery({
    ...pinListOptions(wsId, currentUser?.id ?? ""),
    enabled: !!wsId && !!currentUser?.id,
  });
  const openCreateInitiative = () => useModalStore.getState().open("create-initiative");

  const isWorkspaceAdmin = useMemo(() => {
    if (!currentUser) return false;
    const me = members.find((m: MemberWithUser) => m.user_id === currentUser.id);
    return me?.role === "owner" || me?.role === "admin";
  }, [members, currentUser]);

  const pinnedInitiativeIds = useMemo(() => {
    const s = new Set<string>();
    for (const pin of pins) if (pin.item_type === "initiative") s.add(pin.item_id);
    return s;
  }, [pins]);

  const [search, setSearch] = useState("");
  const [selectedIds, setSelectedIds] = useState<ReadonlySet<string>>(new Set());
  const toggleSelected = (id: string) =>
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const activeFilterCount = countActiveFilters(filters);
  const hasActiveFilters = activeFilterCount > 0;

  const leadOptions = useMemo(() => {
    const m = new Map<string, { type: string; id: string; count: number }>();
    for (const p of initiatives) {
      const v = leadFilterValue(p);
      if (!v || !p.lead_type || !p.lead_id) continue;
      const e = m.get(v);
      if (e) e.count += 1;
      else m.set(v, { type: p.lead_type, id: p.lead_id, count: 1 });
    }
    return m;
  }, [initiatives]);

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    const filtered = initiatives.filter((p) => {
      if (q && !p.title.toLowerCase().includes(q) && !matchesPinyin(p.title, q)) {
        return false;
      }
      if (filters.statuses.length && !filters.statuses.includes(p.status)) return false;
      if (filters.priorities.length && !filters.priorities.includes(p.priority)) {
        return false;
      }
      if (filters.leads.length) {
        const v = leadFilterValue(p);
        if (!v || !filters.leads.includes(v)) return false;
      }
      return true;
    });
    const dir = sortDirection === "asc" ? 1 : -1;
    const sorted = [...filtered];
    sorted.sort((a, b) => {
      if (sortField === "name") return a.title.localeCompare(b.title) * dir;
      if (sortField === "priority") {
        return (
          (PRIORITY_ORDER[a.priority] - PRIORITY_ORDER[b.priority]) * dir ||
          a.title.localeCompare(b.title)
        );
      }
      if (sortField === "status") {
        return (
          (STATUS_ORDER[a.status] - STATUS_ORDER[b.status]) * dir ||
          a.title.localeCompare(b.title)
        );
      }
      if (sortField === "progress") {
        return (progressOf(a) - progressOf(b)) * dir || a.title.localeCompare(b.title);
      }
      return (Date.parse(a.created_at) - Date.parse(b.created_at)) * dir;
    });
    return sorted;
  }, [initiatives, search, filters, sortField, sortDirection]);

  const selectedInitiatives = visible.filter((p) => selectedIds.has(p.id));
  const allSelected = visible.length > 0 && selectedInitiatives.length === visible.length;
  const someSelected = selectedInitiatives.length > 0 && !allSelected;
  const handleToggleAll = () =>
    setSelectedIds(allSelected ? new Set() : new Set(visible.map((p) => p.id)));

  const sortLabel = (f: InitiativeSortField) =>
    f === "name"
      ? t(($) => $.table.name)
      : f === "priority"
        ? t(($) => $.table.priority)
        : f === "status"
          ? t(($) => $.table.status)
          : f === "progress"
            ? t(($) => $.table.progress)
            : t(($) => $.table.created);
  const columnLabel = (k: InitiativeColumnKey) =>
    k === "priority"
      ? t(($) => $.table.priority)
      : k === "progress"
        ? t(($) => $.table.progress)
        : k === "lead"
          ? t(($) => $.table.lead)
          : k === "projects"
            ? t(($) => $.table.projects)
            : t(($) => $.table.created);

  const showEmpty = !isLoading && initiatives.length === 0;
  const countBadge = (n: number) => (
    <span className="ml-auto pl-3 text-caption text-muted-foreground">{n}</span>
  );

  return (
    <div className="relative flex flex-1 min-h-0 flex-col">
      <CollectionPageHeader
        icon={Target}
        title={t(($) => $.page.title)}
        count={initiatives.length}
        actions={
          <CollectionPageHeaderAction
            icon={Plus}
            label={t(($) => $.page.new_initiative)}
            onClick={openCreateInitiative}
          />
        }
      />

      {showEmpty ? (
        <CollectionPageState
          icon={Target}
          title={t(($) => $.page.empty)}
          actions={
            <Button size="sm" variant="outline" onClick={openCreateInitiative}>
              {t(($) => $.page.create_first)}
            </Button>
          }
        />
      ) : (
        <>
          <div className={PAGE_TOOLBAR}>
            <div className="flex min-w-0 items-center gap-2">
              <div className="relative hidden md:block">
                <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  aria-label={t(($) => $.page.search_placeholder)}
                  placeholder={t(($) => $.page.search_placeholder)}
                  className="h-8 w-56 pl-8 text-body"
                />
              </div>
              {(hasActiveFilters || search.trim().length > 0) && (
                <span
                  title={t(($) => $.toolbar.result_count_title)}
                  className="hidden shrink-0 text-caption tabular-nums text-muted-foreground md:inline"
                >
                  {visible.length} / {initiatives.length}
                </span>
              )}
            </div>

            <div className="flex shrink-0 items-center gap-1">
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button
                      variant={hasActiveFilters ? "default" : "outline"}
                      size="sm"
                      className={
                        hasActiveFilters
                          ? "h-8 w-8 gap-1 bg-brand px-0 text-white hover:bg-brand/90 md:w-auto md:px-2.5"
                          : "h-8 w-8 gap-1 px-0 text-muted-foreground md:w-auto md:px-2.5"
                      }
                    >
                      <Filter className="size-3.5" />
                      {hasActiveFilters ? (
                        <>
                          <span className="hidden md:inline">
                            {t(($) => $.toolbar.filter_active_count, { count: activeFilterCount })}
                          </span>
                          <span className="tabular-nums md:hidden">{activeFilterCount}</span>
                        </>
                      ) : (
                        <span className="hidden md:inline">{t(($) => $.toolbar.filter_label)}</span>
                      )}
                      {hasActiveFilters && (
                        <span
                          role="button"
                          tabIndex={-1}
                          aria-label={t(($) => $.toolbar.clear_filters)}
                          className="-mr-1 ml-0.5 hidden rounded-sm p-0.5 hover:bg-white/20 md:inline-flex"
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            clearFilters();
                          }}
                          onPointerDown={(e) => e.stopPropagation()}
                        >
                          <X className="size-3" />
                        </span>
                      )}
                    </Button>
                  }
                />
                <DropdownMenuContent align="end" className="w-auto">
                  <DropdownMenuSub>
                    <DropdownMenuSubTrigger>
                      <span className="flex-1">{t(($) => $.toolbar.section_status)}</span>
                      {filters.statuses.length > 0 && (
                        <span className="text-caption font-medium text-primary">{filters.statuses.length}</span>
                      )}
                    </DropdownMenuSubTrigger>
                    <DropdownMenuSubContent className="w-auto min-w-44">
                      {STATUS_VALUES.map((s) => (
                        <DropdownMenuCheckboxItem
                          key={s}
                          checked={filters.statuses.includes(s)}
                          onCheckedChange={() => toggleFilter("statuses", s)}
                          className={FILTER_ITEM_CLASS}
                        >
                          <HoverCheck checked={filters.statuses.includes(s)} />
                          {t(($) => $.status[s])}
                        </DropdownMenuCheckboxItem>
                      ))}
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>
                  <DropdownMenuSub>
                    <DropdownMenuSubTrigger>
                      <span className="flex-1">{t(($) => $.toolbar.section_priority)}</span>
                      {filters.priorities.length > 0 && (
                        <span className="text-caption font-medium text-primary">{filters.priorities.length}</span>
                      )}
                    </DropdownMenuSubTrigger>
                    <DropdownMenuSubContent className="w-auto min-w-44">
                      {PRIORITY_VALUES.map((pr) => (
                        <DropdownMenuCheckboxItem
                          key={pr}
                          checked={filters.priorities.includes(pr)}
                          onCheckedChange={() => toggleFilter("priorities", pr)}
                          className={FILTER_ITEM_CLASS}
                        >
                          <HoverCheck checked={filters.priorities.includes(pr)} />
                          {t(($) => $.priority[pr])}
                        </DropdownMenuCheckboxItem>
                      ))}
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>
                  <DropdownMenuSub>
                    <DropdownMenuSubTrigger>
                      <span className="flex-1">{t(($) => $.toolbar.section_lead)}</span>
                      {filters.leads.length > 0 && (
                        <span className="text-caption font-medium text-primary">{filters.leads.length}</span>
                      )}
                    </DropdownMenuSubTrigger>
                    <DropdownMenuSubContent className="max-h-72 w-auto min-w-48 overflow-y-auto">
                      {[...leadOptions.entries()].map(([value, { type, id, count }]) => (
                        <DropdownMenuCheckboxItem
                          key={value}
                          checked={filters.leads.includes(value)}
                          onCheckedChange={() => toggleFilter("leads", value)}
                          className={FILTER_ITEM_CLASS}
                        >
                          <HoverCheck checked={filters.leads.includes(value)} />
                          <ActorAvatar actorType={type} actorId={id} size="sm" />
                          <span className="min-w-0 truncate">{getActorName(type, id)}</span>
                          {countBadge(count)}
                        </DropdownMenuCheckboxItem>
                      ))}
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>
                </DropdownMenuContent>
              </DropdownMenu>

              <Popover>
                  <Tooltip>
                    <PopoverTrigger
                      render={
                        <TooltipTrigger
                          render={
                            <Button variant="outline" size="sm" className="h-8 w-8 gap-1 px-0 text-muted-foreground md:w-auto md:px-2.5">
                              {sortDirection === "asc" ? <ArrowUp className="size-3.5" /> : <ArrowDown className="size-3.5" />}
                              <span className="hidden md:inline">{sortLabel(sortField)}</span>
                            </Button>
                          }
                        />
                      }
                    />
                    <TooltipContent side="bottom">{t(($) => $.toolbar.display)}</TooltipContent>
                  </Tooltip>
                  <PopoverContent align="end" className="w-64 p-0">
                    <div className="border-b px-3 py-2.5">
                      <span className="text-caption font-medium text-muted-foreground">{t(($) => $.toolbar.sort_by)}</span>
                      <div className="mt-2 flex items-center gap-1.5">
                        <DropdownMenu>
                          <DropdownMenuTrigger
                            render={
                              <Button variant="outline" size="sm" className="flex-1 justify-between text-caption">
                                {sortLabel(sortField)}
                                <ChevronDown className="size-3 text-muted-foreground" />
                              </Button>
                            }
                          />
                          <DropdownMenuContent align="start" className="w-auto">
                            <DropdownMenuRadioGroup
                              value={sortField}
                              onValueChange={(v) => setSortField(v as InitiativeSortField)}
                            >
                              {SORT_FIELDS.map((f) => (
                                <DropdownMenuRadioItem key={f} value={f}>
                                  {sortLabel(f)}
                                </DropdownMenuRadioItem>
                              ))}
                            </DropdownMenuRadioGroup>
                          </DropdownMenuContent>
                        </DropdownMenu>
                        <Button
                          variant="outline"
                          size="icon-sm"
                          onClick={() => setSortDirection(sortDirection === "asc" ? "desc" : "asc")}
                          title={sortDirection === "asc" ? t(($) => $.toolbar.direction_asc) : t(($) => $.toolbar.direction_desc)}
                        >
                          {sortDirection === "asc" ? <ArrowUp className="size-3.5" /> : <ArrowDown className="size-3.5" />}
                        </Button>
                      </div>
                    </div>
                    {isCompact && (
                      <div className="px-3 py-2.5">
                        <span className="text-caption font-medium text-muted-foreground">{t(($) => $.toolbar.section_columns)}</span>
                        <div className="mt-2 space-y-2">
                          {COLUMN_KEYS.map((key) => (
                            <label key={key} className="flex cursor-pointer items-center justify-between">
                              <span className="text-body">{columnLabel(key)}</span>
                              <Switch size="sm" checked={!hiddenColumns.includes(key)} onCheckedChange={() => toggleColumn(key)} />
                            </label>
                          ))}
                        </div>
                      </div>
                    )}
                  </PopoverContent>
                </Popover>

              <DropdownMenu>
                <Tooltip>
                  <DropdownMenuTrigger
                    render={
                      <TooltipTrigger
                        render={
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-8 w-8 gap-1 px-0 text-muted-foreground md:w-auto md:px-2.5"
                          >
                            {isCompact ? (
                              <Rows3 className="size-3.5" />
                            ) : (
                              <LayoutGrid className="size-3.5" />
                            )}
                            <span className="hidden md:inline">
                              {isCompact ? t(($) => $.page.view_table) : t(($) => $.page.view_cards)}
                            </span>
                          </Button>
                        }
                      />
                    }
                  />
                  <TooltipContent side="bottom">{t(($) => $.toolbar.view)}</TooltipContent>
                </Tooltip>
                <DropdownMenuContent align="end" className="w-auto">
                  <DropdownMenuGroup>
                    <DropdownMenuLabel>{t(($) => $.toolbar.view)}</DropdownMenuLabel>
                  </DropdownMenuGroup>
                  <DropdownMenuRadioGroup
                    value={viewMode}
                    onValueChange={(v) => setViewMode(v as InitiativeViewMode)}
                  >
                    <DropdownMenuRadioItem value="compact">
                      <Rows3 />
                      {t(($) => $.page.view_table)}
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="comfortable">
                      <LayoutGrid />
                      {t(($) => $.page.view_cards)}
                    </DropdownMenuRadioItem>
                  </DropdownMenuRadioGroup>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>

          {isLoading ? (
            <LoadingState isCompact={isCompact} />
          ) : visible.length === 0 ? (
            <div className="flex flex-1 flex-col items-center justify-center py-24 text-muted-foreground">
              <Search className="mb-3 h-10 w-10 opacity-30" />
              <p className="text-body">{t(($) => $.page.no_matches)}</p>
            </div>
          ) : isCompact ? (
            <div className="min-h-0 flex-1 overflow-auto @container">
              <ListGrid
                className={`${GRID_COLS} @2xl:min-w-[var(--inc-minw)]`}
                style={{
                  ...columnTrackVars(isColVisible),
                  paddingBottom: LIST_GRID_BOTTOM_CLEARANCE,
                }}
              >
                <InitiativeTableHeader
                  sortField={sortField}
                  sortDirection={sortDirection}
                  onSort={toggleSort}
                  isColVisible={isColVisible}
                  allSelected={allSelected}
                  someSelected={someSelected}
                  onToggleAll={handleToggleAll}
                />
                {visible.map((initiative) => (
                  <InitiativeTableRow
                    key={initiative.id}
                    initiative={initiative}
                    pinned={pinnedInitiativeIds.has(initiative.id)}
                    canDelete={isWorkspaceAdmin}
                    isColVisible={isColVisible}
                    selected={selectedIds.has(initiative.id)}
                    onToggleSelect={() => toggleSelected(initiative.id)}
                    rowHref={wsPaths.initiativeDetail(initiative.id)}
                    rowLink={rowLink}
                  />
                ))}
              </ListGrid>
            </div>
          ) : (
            <div className={cn("min-h-0 flex-1 overflow-y-auto pt-4", PAGE_GUTTER)}>
              <div
                className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4"
                style={{ paddingBottom: LIST_GRID_BOTTOM_CLEARANCE }}
              >
                {visible.map((initiative) => (
                  <InitiativeCard
                    key={initiative.id}
                    initiative={initiative}
                    pinned={pinnedInitiativeIds.has(initiative.id)}
                    canDelete={isWorkspaceAdmin}
                  />
                ))}
              </div>
            </div>
          )}

          <InitiativeBatchToolbar
            rows={selectedInitiatives}
            pinnedIds={pinnedInitiativeIds}
            canDelete={isWorkspaceAdmin}
            onClear={() => setSelectedIds(new Set())}
          />
        </>
      )}
    </div>
  );
}

function LoadingState({ isCompact }: { isCompact: boolean }) {
  if (isCompact) {
    return (
      <div className={cn("min-h-0 flex-1 overflow-auto pt-4", PAGE_GUTTER)}>
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-11 w-full rounded-md" />
          ))}
        </div>
      </div>
    );
  }
  return (
    <div className={cn("grid grid-cols-1 gap-3 pt-4 sm:grid-cols-2 lg:grid-cols-4", PAGE_GUTTER)}>
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="flex flex-col gap-2 rounded-md border p-3">
          <div className="flex items-center gap-2">
            <Skeleton className="h-8 w-8 rounded" />
            <Skeleton className="h-4 w-3/4" />
          </div>
          <div className="flex gap-1.5">
            <Skeleton className="h-5 w-16 rounded" />
            <Skeleton className="h-5 w-20 rounded" />
          </div>
        </div>
      ))}
    </div>
  );
}
