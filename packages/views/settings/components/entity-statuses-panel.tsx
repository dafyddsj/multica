"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Archive, GripVertical, MoreHorizontal, Pencil, Plus } from "lucide-react";
import { toast } from "sonner";
import {
  DndContext,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useWorkspaceId } from "@multica/core/hooks";
import { useAuthStore } from "@multica/core/auth";
import { memberListOptions } from "@multica/core/workspace/queries";
import {
  ENTITY_STATUS_CATEGORIES,
  entityStatusColor,
  entityStatusListOptions,
} from "@multica/core/entity-statuses";
import {
  useArchiveEntityStatus,
  useCreateEntityStatus,
  useReorderEntityStatuses,
  useUpdateEntityStatus,
} from "@multica/core/entity-statuses/mutations";
import type {
  EntityStatusCategory,
  EntityStatusEntry,
  EntityStatusResourceType,
} from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { Textarea } from "@multica/ui/components/ui/textarea";
import { Label as FieldLabel } from "@multica/ui/components/ui/label";
import { Switch } from "@multica/ui/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@multica/ui/components/ui/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@multica/ui/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@multica/ui/components/ui/tooltip";
import { ColorPicker, COLOR_PICKER_PRESETS } from "../../common/color-picker";
import { useT } from "../../i18n";

interface StatusDraft {
  name: string;
  description: string;
  category: EntityStatusCategory;
  color: string;
}

const EMPTY_DRAFT: StatusDraft = {
  name: "",
  description: "",
  category: "planned",
  color: COLOR_PICKER_PRESETS[6]!,
};

export function EntityStatusesPanel({
  resourceType,
}: {
  resourceType: EntityStatusResourceType;
}) {
  const { t } = useT("settings");
  const wsId = useWorkspaceId();
  const [showArchived, setShowArchived] = useState(false);
  const [createCategory, setCreateCategory] = useState<EntityStatusCategory | null>(null);
  const [editing, setEditing] = useState<EntityStatusEntry | null>(null);
  const [pendingArchive, setPendingArchive] = useState<EntityStatusEntry | null>(null);

  const { data: statuses = [], isLoading } = useQuery(
    entityStatusListOptions(wsId, resourceType),
  );
  const { data: members = [] } = useQuery(memberListOptions(wsId));
  const currentUser = useAuthStore((s) => s.user);
  const myRole = useMemo(() => {
    if (!currentUser) return null;
    return members.find((m) => m.user_id === currentUser.id)?.role ?? null;
  }, [members, currentUser]);
  const isAdmin = myRole === "owner" || myRole === "admin";

  const groups = useMemo(
    () =>
      ENTITY_STATUS_CATEGORIES.map((category) => {
        const inCategory = statuses.filter((s) => s.category === category);
        return {
          category,
          builtIn: inCategory.find((s) => s.is_system),
          custom: inCategory.filter(
            (s) => !s.is_system && (showArchived || !s.archived_at),
          ),
        };
      }),
    [statuses, showArchived],
  );

  const archivedCount = statuses.filter((s) => !s.is_system && s.archived_at).length;
  const scopeLabel = t(($) => $.statuses.scopes[resourceType]);

  return (
    <div className="space-y-4">
      {archivedCount > 0 && (
        <label className="flex items-center justify-end gap-2 text-caption text-muted-foreground">
          {t(($) => $.statuses.show_archived, { count: archivedCount })}
          <Switch checked={showArchived} onCheckedChange={setShowArchived} />
        </label>
      )}

      {isLoading ? (
        <div className="rounded-lg border border-surface-border bg-card px-4 py-12 text-center text-body text-muted-foreground">
          {t(($) => $.statuses.loading)}
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-surface-border bg-card">
          {groups.map((group) => (
            <CategorySection
              key={group.category}
              resourceType={resourceType}
              category={group.category}
              builtIn={group.builtIn}
              custom={group.custom}
              canManage={isAdmin}
              onCreate={() => setCreateCategory(group.category)}
              onEdit={setEditing}
              onArchive={setPendingArchive}
            />
          ))}
        </div>
      )}

      <StatusEditorDialog
        resourceType={resourceType}
        open={createCategory !== null}
        onOpenChange={(open) => !open && setCreateCategory(null)}
        category={createCategory}
      />
      <StatusEditorDialog
        resourceType={resourceType}
        open={Boolean(editing)}
        onOpenChange={(open) => !open && setEditing(null)}
        category={editing?.category ?? null}
        status={editing}
      />
      <ArchiveStatusDialog
        resourceType={resourceType}
        scopeLabel={scopeLabel}
        status={pendingArchive}
        onClose={() => setPendingArchive(null)}
      />
    </div>
  );
}

function CategorySection({
  resourceType,
  category,
  builtIn,
  custom,
  canManage,
  onCreate,
  onEdit,
  onArchive,
}: {
  resourceType: EntityStatusResourceType;
  category: EntityStatusCategory;
  builtIn: EntityStatusEntry | undefined;
  custom: EntityStatusEntry[];
  canManage: boolean;
  onCreate: () => void;
  onEdit: (status: EntityStatusEntry) => void;
  onArchive: (status: EntityStatusEntry) => void;
}) {
  const { t } = useT("settings");
  const reorder = useReorderEntityStatuses(resourceType);
  const [order, setOrder] = useState(custom);
  useEffect(() => setOrder(custom), [custom]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const from = order.findIndex((s) => s.id === active.id);
    const to = order.findIndex((s) => s.id === over.id);
    if (from < 0 || to < 0) return;
    const next = arrayMove(order, from, to);
    setOrder(next);
    reorder.mutate(
      { category, ordered: next.filter((entry) => !entry.archived_at) },
      {
        onError: (error) => {
          setOrder(custom);
          toast.error(
            error instanceof Error ? error.message : t(($) => $.statuses.reorder_failed),
          );
        },
      },
    );
  };

  const sortableIds = order.filter((s) => !s.archived_at).map((s) => s.id);
  const canReorder = canManage && sortableIds.length > 1;

  return (
    <section className="border-b border-surface-border last:border-b-0">
      <div className="flex items-center justify-between gap-2 bg-muted/20 px-4 py-1.5">
        <span className="text-caption font-medium text-muted-foreground">
          {builtIn?.name ?? t(($) => $.statuses.category_labels[category])}
        </span>
        {canManage && (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t(($) => $.statuses.add)}
                  onClick={onCreate}
                >
                  <Plus className="size-4" />
                </Button>
              }
            />
            <TooltipContent>{t(($) => $.statuses.add)}</TooltipContent>
          </Tooltip>
        )}
      </div>

      <div className="divide-y divide-surface-border">
        {builtIn && (
          <BuiltInRow
            entry={builtIn}
            behavior={t(($) => $.statuses.categories[category])}
            canManage={canManage}
            onEdit={() => onEdit(builtIn)}
          />
        )}
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={sortableIds} strategy={verticalListSortingStrategy}>
            {order.map((entry) => (
              <CustomStatusRow
                key={entry.id}
                entry={entry}
                canManage={canManage}
                canReorder={canReorder && !entry.archived_at}
                onEdit={() => onEdit(entry)}
                onArchive={() => onArchive(entry)}
              />
            ))}
          </SortableContext>
        </DndContext>
      </div>
    </section>
  );
}

function StatusDot({ color, className }: { color: string; className?: string }) {
  return (
    <span
      className={className ?? "size-4 rounded-full"}
      style={{ backgroundColor: color }}
    />
  );
}

function BuiltInRow({
  entry,
  behavior,
  canManage,
  onEdit,
}: {
  entry: EntityStatusEntry;
  behavior: string;
  canManage: boolean;
  onEdit: () => void;
}) {
  const { t } = useT("settings");
  return (
    <div className="flex min-h-12 items-center gap-3 px-4 py-2">
      <StatusDot color={entry.color} className="size-3.5 shrink-0 rounded-full" />
      <div className="min-w-0 flex-1">
        <p className="truncate text-body font-medium">{entry.name}</p>
        <p className="truncate text-caption text-muted-foreground">{behavior}</p>
      </div>
      {canManage && (
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label={t(($) => $.statuses.actions.open, { name: entry.name })}
          onClick={onEdit}
        >
          <Pencil className="size-4" />
        </Button>
      )}
    </div>
  );
}

function CustomStatusRow({
  entry,
  canManage,
  canReorder,
  onEdit,
  onArchive,
}: {
  entry: EntityStatusEntry;
  canManage: boolean;
  canReorder: boolean;
  onEdit: () => void;
  onArchive: () => void;
}) {
  const { t } = useT("settings");
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: entry.id,
    disabled: !canReorder,
  });
  const archived = Boolean(entry.archived_at);
  const color = entityStatusColor(entry) ?? entry.color;

  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={`group/row relative flex min-h-12 items-center gap-3 bg-card px-4 py-2 ${isDragging ? "z-10 shadow-[var(--surface-shadow)]" : ""} ${archived ? "opacity-60" : ""}`}
    >
      {canReorder && (
        <button
          type="button"
          aria-label={t(($) => $.statuses.actions.reorder, { name: entry.name })}
          className="absolute left-0 top-1/2 flex w-4 -translate-y-1/2 cursor-grab justify-center text-faint-foreground opacity-0 transition-opacity group-hover/row:opacity-100 focus-visible:opacity-100 active:cursor-grabbing"
          {...attributes}
          {...listeners}
        >
          <GripVertical className="size-4" />
        </button>
      )}
      <StatusDot color={color} className="size-3.5 shrink-0 rounded-full" />
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate text-body font-medium">{entry.name}</span>
          {archived && (
            <span className="shrink-0 rounded-full bg-muted/60 px-1.5 py-0.5 text-micro text-muted-foreground">
              {t(($) => $.statuses.archived_badge)}
            </span>
          )}
        </div>
        {entry.description && (
          <p className="truncate text-caption text-muted-foreground">{entry.description}</p>
        )}
      </div>
      {canManage && !archived && (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t(($) => $.statuses.actions.open, { name: entry.name })}
              >
                <MoreHorizontal className="size-4" />
              </Button>
            }
          />
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onEdit}>
              <Pencil className="size-4" />
              {t(($) => $.statuses.actions.edit)}
            </DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onClick={onArchive}>
              <Archive className="size-4" />
              {t(($) => $.statuses.actions.archive)}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}

function StatusEditorDialog({
  resourceType,
  open,
  onOpenChange,
  category,
  status,
}: {
  resourceType: EntityStatusResourceType;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  category: EntityStatusCategory | null;
  status?: EntityStatusEntry | null;
}) {
  const { t } = useT("settings");
  const create = useCreateEntityStatus(resourceType);
  const update = useUpdateEntityStatus(resourceType);
  const [draft, setDraft] = useState<StatusDraft>(EMPTY_DRAFT);

  const categoryItems = ENTITY_STATUS_CATEGORIES.map((c) => ({
    value: c,
    label: t(($) => $.statuses.category_labels[c]),
  }));

  useEffect(() => {
    if (!open) return;
    setDraft(
      status
        ? {
            name: status.name,
            description: status.description ?? "",
            category: status.category,
            color: status.color,
          }
        : { ...EMPTY_DRAFT, category: category ?? "planned" },
    );
  }, [status, category, open]);

  const submit = () => {
    const name = draft.name.trim();
    if (!name) return;
    const onError = (error: unknown) =>
      toast.error(
        error instanceof Error ? error.message : t(($) => $.statuses.editor.save_failed),
      );

    if (status) {
      update.mutate(
        {
          id: status.id,
          name,
          description: draft.description.trim(),
          color: draft.color,
        },
        { onSuccess: () => onOpenChange(false), onError },
      );
      return;
    }
    create.mutate(
      {
        name,
        description: draft.description.trim(),
        category: draft.category,
        color: draft.color,
      },
      { onSuccess: () => onOpenChange(false), onError },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {status
              ? t(($) => $.statuses.editor.edit_title)
              : t(($) => $.statuses.editor.create_title)}
          </DialogTitle>
          <DialogDescription>
            {t(($) => $.statuses.editor.behavior_hint, {
              category: t(($) => $.statuses.category_labels[draft.category]),
            })}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-5 py-2">
          <div className="space-y-2">
            <FieldLabel htmlFor="entity-status-name">
              {t(($) => $.statuses.editor.name)}
            </FieldLabel>
            <Input
              id="entity-status-name"
              autoFocus
              maxLength={64}
              value={draft.name}
              onChange={(event) =>
                setDraft((current) => ({ ...current, name: event.target.value }))
              }
              placeholder={t(($) => $.statuses.editor.name_placeholder)}
            />
            {status && (
              <p className="text-caption text-muted-foreground">
                {t(($) => $.statuses.editor.key_hint, { key: status.key })}
              </p>
            )}
          </div>
          {!status?.is_system && (
            <div className="space-y-2">
              <FieldLabel>{t(($) => $.statuses.editor.category)}</FieldLabel>
              <Select
                items={categoryItems}
                value={draft.category}
                onValueChange={(value) =>
                  value &&
                  setDraft((current) => ({
                    ...current,
                    category: value as EntityStatusCategory,
                  }))
                }
                disabled={Boolean(status)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {categoryItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-caption text-muted-foreground">
                {status
                  ? t(($) => $.statuses.editor.category_locked)
                  : t(($) => $.statuses.categories[draft.category])}
              </p>
            </div>
          )}
          <div className="space-y-2">
            <FieldLabel htmlFor="entity-status-description">
              {t(($) => $.statuses.editor.description)}
            </FieldLabel>
            <Textarea
              id="entity-status-description"
              rows={3}
              maxLength={256}
              value={draft.description}
              onChange={(event) =>
                setDraft((current) => ({ ...current, description: event.target.value }))
              }
              placeholder={t(($) => $.statuses.editor.description_placeholder)}
            />
          </div>
          <div className="space-y-2">
            <FieldLabel>{t(($) => $.statuses.editor.color)}</FieldLabel>
            <ColorPicker
              value={draft.color}
              onChange={(color) => setDraft((current) => ({ ...current, color }))}
              trigger={
                <button
                  type="button"
                  aria-label={t(($) => $.statuses.editor.color)}
                  className="flex h-9 items-center gap-2.5 rounded-md border border-surface-border px-2.5 transition-colors hover:bg-surface-hover"
                >
                  <span className="size-5 rounded-full" style={{ backgroundColor: draft.color }} />
                  <span className="font-mono text-caption uppercase text-muted-foreground">
                    {draft.color}
                  </span>
                </button>
              }
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {t(($) => $.statuses.editor.cancel)}
          </Button>
          <Button
            onClick={submit}
            disabled={!draft.name.trim() || create.isPending || update.isPending}
          >
            {create.isPending || update.isPending
              ? t(($) => $.statuses.editor.saving)
              : t(($) => $.statuses.editor.save)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ArchiveStatusDialog({
  resourceType,
  scopeLabel,
  status,
  onClose,
}: {
  resourceType: EntityStatusResourceType;
  scopeLabel: string;
  status: EntityStatusEntry | null;
  onClose: () => void;
}) {
  const { t } = useT("settings");
  const archive = useArchiveEntityStatus(resourceType);
  return (
    <AlertDialog open={Boolean(status)} onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t(($) => $.statuses.archive_dialog.title)}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(($) => $.statuses.archive_dialog.description, {
              name: status?.name ?? "",
              scope: scopeLabel,
            })}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t(($) => $.statuses.archive_dialog.cancel)}</AlertDialogCancel>
          <AlertDialogAction
            onClick={() => {
              if (!status) return;
              archive.mutate(status.id, {
                onSuccess: onClose,
                onError: (error) =>
                  toast.error(
                    error instanceof Error
                      ? error.message
                      : t(($) => $.statuses.archive_dialog.failed),
                  ),
              });
            }}
          >
            {t(($) => $.statuses.archive_dialog.confirm)}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
