"use client";

import { useMemo, useState, useCallback, useRef, useEffect } from "react";
import { useDefaultLayout, usePanelRef } from "react-resizable-panels";
import {
  Check,
  ChevronRight,
  FolderKanban,
  Link2,
  MoreHorizontal,
  PanelRight,
  Pin,
  PinOff,
  Plus,
  Search,
  Trash2,
  UserMinus,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { cn } from "@multica/ui/lib/utils";
import { copyText } from "@multica/ui/lib/clipboard";
import { toast } from "sonner";
import type { InitiativeStatus, InitiativePriority, Project } from "@multica/core/types";
import { useAuthStore } from "@multica/core/auth";
import { initiativeDetailOptions } from "@multica/core/initiatives/queries";
import { useUpdateInitiative, useDeleteInitiative } from "@multica/core/initiatives/mutations";
import { INITIATIVE_STATUS_ORDER, INITIATIVE_STATUS_CONFIG, INITIATIVE_PRIORITY_ORDER } from "@multica/core/initiatives/config";
import { projectListOptions } from "@multica/core/projects/queries";
import { useUpdateProject } from "@multica/core/projects/mutations";
import { pinListOptions } from "@multica/core/pins";
import { useCreatePin, useDeletePin } from "@multica/core/pins";
import { memberListOptions, agentListOptions } from "@multica/core/workspace/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { useActorName } from "@multica/core/workspace/hooks";
import { useModalStore } from "@multica/core/modals";
import { ActorAvatar } from "../../common/actor-avatar";
import { useNavigation, useRowLink } from "../../navigation";
import { TitleEditor, ContentEditor, type ContentEditorRef } from "../../editor";
import { PriorityIcon } from "../../issues/components/priority-icon";
import { ProjectStartDatePicker } from "../../projects/components/project-start-date-picker";
import { ProjectDueDatePicker } from "../../projects/components/project-due-date-picker";
import { ProjectIcon } from "../../projects/components/project-icon";
import { PROJECT_STATUS_CONFIG } from "@multica/core/projects/config";
import { useProjectStatusLabels } from "../../projects/components/labels";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from "@multica/ui/components/ui/resizable";
import { Sheet, SheetContent } from "@multica/ui/components/ui/sheet";
import { useIsMobile } from "@multica/ui/hooks/use-mobile";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "@multica/ui/components/ui/popover";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@multica/ui/components/ui/tooltip";
import { EmojiPicker } from "@multica/ui/components/common/emoji-picker";
import { BreadcrumbHeader } from "../../layout/breadcrumb-header";
import {
  AnimatedRightSidebar,
  getAnimatedRightSidebarInitialOpen,
  rightSidebarPanelMotionProps,
  useAnimatedRightSidebarState,
  useRightSidebarShortcut,
} from "../../layout/animated-right-sidebar";
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
import { useT } from "../../i18n";
import { useInitiativeStatusLabels, useInitiativePriorityLabels } from "./labels";
import { matchesPinyin } from "../../editor/extensions/pinyin-match";
import { PAGE_GUTTER } from "../../layout/page-header";

function PropRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-8 items-center gap-2 rounded-md px-2 -mx-2 hover:bg-accent/50 transition-colors">
      <span className="w-16 shrink-0 text-caption text-muted-foreground">{label}</span>
      <div className="flex min-w-0 flex-1 items-center gap-1.5 text-caption truncate">
        {children}
      </div>
    </div>
  );
}

function ChildProjectRow({
  project,
  href,
}: {
  project: Project;
  href: string;
}) {
  const statusLabels = useProjectStatusLabels();
  const rowLink = useRowLink();
  const progress =
    project.issue_count > 0
      ? Math.round((project.done_count / project.issue_count) * 100)
      : null;

  return (
    <div
      className="group/row flex h-11 cursor-pointer items-center gap-3 rounded-md px-2 hover:bg-accent/50"
      {...rowLink(href, project.title)}
    >
      <ProjectIcon project={project} size="sm" />
      <span className="min-w-0 flex-1 truncate text-body font-medium">{project.title}</span>
      <span
        className={cn(
          "inline-flex items-center rounded px-1.5 py-0.5 text-caption font-medium",
          PROJECT_STATUS_CONFIG[project.status].badgeBg,
          PROJECT_STATUS_CONFIG[project.status].badgeText,
        )}
      >
        {statusLabels[project.status]}
      </span>
      {progress === null ? (
        <span className="w-16 text-right text-caption text-faint-foreground">—</span>
      ) : (
        <span className="w-16 text-right text-caption tabular-nums text-muted-foreground">
          {project.done_count}/{project.issue_count}
        </span>
      )}
    </div>
  );
}

export function InitiativeDetail({ initiativeId }: { initiativeId: string }) {
  const { t } = useT("initiatives");
  const statusLabels = useInitiativeStatusLabels();
  const priorityLabels = useInitiativePriorityLabels();
  const wsId = useWorkspaceId();
  const wsPaths = useWorkspacePaths();
  const router = useNavigation();
  const userId = useAuthStore((s) => s.user?.id);
  const { data: initiative, isLoading } = useQuery(initiativeDetailOptions(wsId, initiativeId));
  const { data: allProjects = [] } = useQuery(projectListOptions(wsId));
  const { data: members = [] } = useQuery(memberListOptions(wsId));
  const { data: agents = [] } = useQuery(agentListOptions(wsId));
  const { getActorName } = useActorName();
  const updateInitiative = useUpdateInitiative();
  const deleteInitiative = useDeleteInitiative();
  const updateProject = useUpdateProject();
  const { data: pinnedItems = [] } = useQuery({
    ...pinListOptions(wsId, userId ?? ""),
    enabled: !!userId,
  });
  const isPinned = pinnedItems.some((p) => p.item_type === "initiative" && p.item_id === initiativeId);
  const isWorkspaceAdmin = useMemo(() => {
    if (!userId) return false;
    const me = members.find((m) => m.user_id === userId);
    return me?.role === "owner" || me?.role === "admin";
  }, [members, userId]);
  const createPin = useCreatePin();
  const deletePinMut = useDeletePin();
  const descEditorRef = useRef<ContentEditorRef>(null);
  const isMobile = useIsMobile();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [iconPickerOpen, setIconPickerOpen] = useState(false);
  const [propertiesOpen, setPropertiesOpen] = useState(true);
  const [progressOpen, setProgressOpen] = useState(true);
  const [descriptionOpen, setDescriptionOpen] = useState(true);
  const [attachOpen, setAttachOpen] = useState(false);
  const [attachFilter, setAttachFilter] = useState("");

  const { defaultLayout, onLayoutChanged } = useDefaultLayout({
    id: "multica_initiative_detail_layout",
  });
  const sidebarRef = usePanelRef();
  const rightSidebarShortcutTargetRef = useRef<HTMLDivElement | null>(null);
  const desktopSidebarInitialOpen = getAnimatedRightSidebarInitialOpen(
    true,
    defaultLayout,
  );
  const {
    open: desktopSidebarOpen,
    visualOpen: desktopSidebarVisualOpen,
    motionEnabled: desktopSidebarMotionEnabled,
    beginToggle: beginDesktopSidebarToggle,
    handleResize: handleDesktopSidebarResize,
  } = useAnimatedRightSidebarState(desktopSidebarInitialOpen);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const sidebarOpen = isMobile ? mobileSidebarOpen : desktopSidebarOpen;

  useEffect(() => {
    if (isMobile) {
      setMobileSidebarOpen(false);
    }
  }, [isMobile]);

  const handleToggleSidebar = useCallback(() => {
    if (isMobile) {
      setMobileSidebarOpen((open) => !open);
      return;
    }

    const panel = sidebarRef.current;
    if (!panel) return;
    const nextOpen = panel.isCollapsed();
    beginDesktopSidebarToggle(nextOpen);
    window.requestAnimationFrame(() => {
      if (nextOpen) panel.expand();
      else panel.collapse();
    });
  }, [beginDesktopSidebarToggle, isMobile, sidebarRef]);

  useRightSidebarShortcut(rightSidebarShortcutTargetRef, handleToggleSidebar);

  const [leadOpen, setLeadOpen] = useState(false);
  const [leadFilter, setLeadFilter] = useState("");
  const leadQuery = leadFilter.toLowerCase();
  const filteredMembers = members.filter((m) => m.name.toLowerCase().includes(leadQuery) || matchesPinyin(m.name, leadQuery));
  const filteredAgents = agents.filter((a) => !a.archived_at && (a.name.toLowerCase().includes(leadQuery) || matchesPinyin(a.name, leadQuery)));

  const childProjects = useMemo(
    () => allProjects.filter((p) => p.initiative_id === initiativeId),
    [allProjects, initiativeId],
  );
  const attachableProjects = useMemo(() => {
    const q = attachFilter.trim().toLowerCase();
    return allProjects.filter((p) => {
      if (p.initiative_id === initiativeId) return false;
      if (!q) return true;
      return p.title.toLowerCase().includes(q) || matchesPinyin(p.title, q);
    });
  }, [allProjects, attachFilter, initiativeId]);

  const handleUpdateField = useCallback(
    (data: Parameters<typeof updateInitiative.mutate>[0] extends { id: string } & infer R ? R : never) => {
      if (!initiative) return;
      updateInitiative.mutate({ id: initiative.id, ...data });
    },
    [initiative, updateInitiative],
  );

  const handleDelete = useCallback(() => {
    if (!initiative) return;
    deleteInitiative.mutate(initiative.id, {
      onSuccess: () => {
        toast.success(t(($) => $.detail.toast_initiative_deleted));
        router.push(wsPaths.initiatives());
      },
    });
  }, [initiative, deleteInitiative, router, wsPaths, t]);

  const handleAttach = useCallback(
    (projectId: string) => {
      updateProject.mutate(
        { id: projectId, initiative_id: initiativeId },
        {
          onSuccess: () => {
            toast.success(t(($) => $.detail.toast_project_attached));
            setAttachOpen(false);
            setAttachFilter("");
          },
          onError: (err) =>
            toast.error(
              err instanceof Error ? err.message : t(($) => $.detail.toast_project_attach_failed),
            ),
        },
      );
    },
    [initiativeId, t, updateProject],
  );

  const openCreateProject = () =>
    useModalStore.getState().open("create-project", { initiative_id: initiativeId });

  if (isLoading) {
    return (
      <div className="mx-auto w-full max-w-4xl px-8 py-10 space-y-4">
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-96" />
        <Skeleton className="h-40 w-full mt-8" />
      </div>
    );
  }

  if (!initiative) {
    return <div className="flex items-center justify-center h-full text-muted-foreground">{t(($) => $.detail.not_found)}</div>;
  }

  const statusCfg = INITIATIVE_STATUS_CONFIG[initiative.status];
  const totalCount = initiative.issue_count;
  const completedCount = initiative.done_count;

  const sidebarContent = (
    <div className="space-y-5">
      <div>
        <Popover open={iconPickerOpen} onOpenChange={setIconPickerOpen}>
          <PopoverTrigger
            render={
              <button
                type="button"
                className="text-display-sm cursor-pointer rounded-lg p-1 -ml-1 hover:bg-accent/60 transition-colors"
                title={t(($) => $.detail.icon_tooltip)}
              >
                {initiative.icon || "🎯"}
              </button>
            }
          />
          <PopoverContent align="start" className="w-auto p-0">
            <EmojiPicker
              onSelect={(emoji) => {
                handleUpdateField({ icon: emoji });
                setIconPickerOpen(false);
              }}
            />
          </PopoverContent>
        </Popover>
        <TitleEditor
          key={`title-${initiativeId}`}
          defaultValue={initiative.title}
          placeholder={t(($) => $.detail.title_placeholder)}
          className="mt-2 w-full text-title-sm font-semibold leading-snug tracking-tight"
          onBlur={(value) => {
            const trimmed = value.trim();
            if (trimmed && trimmed !== initiative.title) handleUpdateField({ title: trimmed });
          }}
        />
      </div>

      <div>
        <button
          type="button"
          className={`flex w-full items-center gap-1 rounded-md px-2 py-1 text-caption font-medium transition-colors mb-2 hover:bg-accent/70 ${propertiesOpen ? "" : "text-muted-foreground hover:text-foreground"}`}
          onClick={() => setPropertiesOpen(!propertiesOpen)}
        >
          {t(($) => $.detail.section_properties)}
          <ChevronRight className={`!size-3 shrink-0 stroke-[2.5] text-muted-foreground transition-transform ${propertiesOpen ? "rotate-90" : ""}`} />
        </button>
        {propertiesOpen && <div className="space-y-0.5 pl-2">
          <PropRow label={t(($) => $.table.status)}>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <button type="button" className="inline-flex items-center gap-1.5 text-caption hover:text-foreground transition-colors">
                    <span className={cn("size-2 rounded-full", statusCfg.dotColor)} />
                    <span>{statusLabels[initiative.status]}</span>
                  </button>
                }
              />
              <DropdownMenuContent align="start" className="w-44">
                {INITIATIVE_STATUS_ORDER.map((s) => (
                  <DropdownMenuItem key={s} onClick={() => handleUpdateField({ status: s as InitiativeStatus })}>
                    <span className={cn("size-2 rounded-full", INITIATIVE_STATUS_CONFIG[s].dotColor)} />
                    <span>{statusLabels[s]}</span>
                    {s === initiative.status && <Check className="ml-auto h-3.5 w-3.5" />}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </PropRow>
          <PropRow label={t(($) => $.table.priority)}>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <button type="button" className="inline-flex items-center gap-1.5 text-caption hover:text-foreground transition-colors">
                    <PriorityIcon priority={initiative.priority} />
                    <span>{priorityLabels[initiative.priority]}</span>
                  </button>
                }
              />
              <DropdownMenuContent align="start" className="w-44">
                {INITIATIVE_PRIORITY_ORDER.map((p) => (
                  <DropdownMenuItem key={p} onClick={() => handleUpdateField({ priority: p as InitiativePriority })}>
                    <PriorityIcon priority={p} />
                    <span>{priorityLabels[p]}</span>
                    {p === initiative.priority && <Check className="ml-auto h-3.5 w-3.5" />}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </PropRow>
          <PropRow label={t(($) => $.table.lead)}>
            <Popover open={leadOpen} onOpenChange={(v) => { setLeadOpen(v); if (!v) setLeadFilter(""); }}>
              <PopoverTrigger
                render={
                  <button type="button" className="inline-flex items-center gap-1.5 text-caption hover:text-foreground transition-colors">
                    {initiative.lead_type && initiative.lead_id ? (
                      <>
                        <ActorAvatar actorType={initiative.lead_type} actorId={initiative.lead_id} size="sm" enableHoverCard showStatusDot />
                        <span className="cursor-pointer">{getActorName(initiative.lead_type, initiative.lead_id)}</span>
                      </>
                    ) : (
                      <span className="text-muted-foreground">{t(($) => $.lead.no_lead)}</span>
                    )}
                  </button>
                }
              />
              <PopoverContent align="start" className="w-52 p-0">
                <div className="px-2 py-1.5 border-b">
                  <input
                    type="text"
                    value={leadFilter}
                    onChange={(e) => setLeadFilter(e.target.value)}
                    placeholder={t(($) => $.lead.assign_placeholder)}
                    className="w-full bg-transparent text-body placeholder:text-muted-foreground outline-none"
                  />
                </div>
                <div className="p-1 max-h-60 overflow-y-auto">
                  <button
                    type="button"
                    onClick={() => { handleUpdateField({ lead_type: null, lead_id: null }); setLeadOpen(false); }}
                    className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-body hover:bg-accent transition-colors"
                  >
                    <UserMinus className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="text-muted-foreground">{t(($) => $.lead.no_lead)}</span>
                  </button>
                  {filteredMembers.length > 0 && (
                    <>
                      <div className="px-2 pt-2 pb-1 text-caption font-medium text-muted-foreground uppercase tracking-wider">{t(($) => $.lead.members_group)}</div>
                      {filteredMembers.map((m) => (
                        <button
                          type="button"
                          key={m.user_id}
                          onClick={() => { handleUpdateField({ lead_type: "member", lead_id: m.user_id }); setLeadOpen(false); }}
                          className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-body hover:bg-accent transition-colors"
                        >
                          <ActorAvatar actorType="member" actorId={m.user_id} size="sm" />
                          <span>{m.name}</span>
                        </button>
                      ))}
                    </>
                  )}
                  {filteredAgents.length > 0 && (
                    <>
                      <div className="px-2 pt-2 pb-1 text-caption font-medium text-muted-foreground uppercase tracking-wider">{t(($) => $.lead.agents_group)}</div>
                      {filteredAgents.map((a) => (
                        <button
                          type="button"
                          key={a.id}
                          onClick={() => { handleUpdateField({ lead_type: "agent", lead_id: a.id }); setLeadOpen(false); }}
                          className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-body hover:bg-accent transition-colors"
                        >
                          <ActorAvatar actorType="agent" actorId={a.id} size="sm" showStatusDot />
                          <span>{a.name}</span>
                        </button>
                      ))}
                    </>
                  )}
                  {filteredMembers.length === 0 && filteredAgents.length === 0 && leadFilter && (
                    <div className="px-2 py-3 text-center text-body text-muted-foreground">{t(($) => $.lead.no_results)}</div>
                  )}
                </div>
              </PopoverContent>
            </Popover>
          </PropRow>
          <PropRow label={t(($) => $.detail.prop_start_date)}>
            <ProjectStartDatePicker startDate={initiative.start_date} onUpdate={handleUpdateField} />
          </PropRow>
          <PropRow label={t(($) => $.detail.prop_due_date)}>
            <ProjectDueDatePicker dueDate={initiative.due_date} onUpdate={handleUpdateField} />
          </PropRow>
        </div>}
      </div>

      {totalCount > 0 && (() => {
        const pct = Math.round((completedCount / totalCount) * 100);
        return (
          <div>
            <button
              type="button"
              className={`flex w-full items-center gap-1 rounded-md px-2 py-1 text-caption font-medium transition-colors mb-2 hover:bg-accent/70 ${progressOpen ? "" : "text-muted-foreground hover:text-foreground"}`}
              onClick={() => setProgressOpen(!progressOpen)}
            >
              {t(($) => $.detail.section_progress)}
              <ChevronRight className={`!size-3 shrink-0 stroke-[2.5] text-muted-foreground transition-transform ${progressOpen ? "rotate-90" : ""}`} />
            </button>
            {progressOpen && <div className="pl-2 flex items-center gap-3">
              <div className="relative h-2 flex-1 rounded-full bg-muted overflow-hidden">
                <div
                  className="absolute inset-y-0 left-0 rounded-full bg-emerald-500 transition-all"
                  style={{ width: `${pct}%` }}
                />
              </div>
              <span className="text-caption text-muted-foreground tabular-nums shrink-0">
                {completedCount}/{totalCount}
              </span>
            </div>}
          </div>
        );
      })()}

      <div>
        <button
          type="button"
          className={`flex w-full items-center gap-1 rounded-md px-2 py-1 text-caption font-medium transition-colors mb-2 hover:bg-accent/70 ${descriptionOpen ? "" : "text-muted-foreground hover:text-foreground"}`}
          onClick={() => setDescriptionOpen(!descriptionOpen)}
        >
          {t(($) => $.detail.section_description)}
          <ChevronRight className={`!size-3 shrink-0 stroke-[2.5] text-muted-foreground transition-transform ${descriptionOpen ? "rotate-90" : ""}`} />
        </button>
        {descriptionOpen && <div className="pl-2">
          <ContentEditor
            ref={descEditorRef}
            key={initiativeId}
            value={initiative.description || ""}
            placeholder={t(($) => $.detail.description_placeholder)}
            onUpdate={(md) => handleUpdateField({ description: md || null })}
            debounceMs={1500}
          />
          <p className="mt-1 px-2 text-caption text-muted-foreground">
            {t(($) => $.detail.description_hint)}
          </p>
        </div>}
      </div>
    </div>
  );

  return (
    <>
    <ResizablePanelGroup orientation="horizontal" className="flex-1 min-h-0" defaultLayout={defaultLayout} onLayoutChanged={onLayoutChanged}>
      <ResizablePanel id="content" minSize="50%">
        <div ref={rightSidebarShortcutTargetRef} className="flex h-full flex-col">
          <BreadcrumbHeader
            segments={[{ href: wsPaths.initiatives(), label: t(($) => $.detail.breadcrumb_fallback) }]}
            leaf={<span className="truncate font-medium text-foreground">{initiative.title}</span>}
            actions={
              <>
              <Button
                variant="ghost"
                size="icon-sm"
                className={cn("text-muted-foreground", isPinned && "text-foreground")}
                title={isPinned ? t(($) => $.detail.unpin_tooltip) : t(($) => $.detail.pin_tooltip)}
                onClick={() => {
                  if (isPinned) {
                    deletePinMut.mutate({ itemType: "initiative", itemId: initiativeId });
                  } else {
                    createPin.mutate({ item_type: "initiative", item_id: initiativeId });
                  }
                }}
              >
                {isPinned ? <PinOff /> : <Pin />}
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button variant="ghost" size="icon-sm" className="text-muted-foreground">
                      <MoreHorizontal />
                    </Button>
                  }
                />
                <DropdownMenuContent align="end" className="w-auto">
                  <DropdownMenuItem onClick={() => {
                    void copyText(window.location.href).then((ok) => {
                      if (ok) toast.success(t(($) => $.detail.toast_link_copied));
                    });
                  }}>
                    <Link2 className="h-3.5 w-3.5" />
                    {t(($) => $.detail.copy_link)}
                  </DropdownMenuItem>
                  {isWorkspaceAdmin && (
                    <>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        variant="destructive"
                        onClick={() => setDeleteDialogOpen(true)}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                        {t(($) => $.detail.delete_action)}
                      </DropdownMenuItem>
                    </>
                  )}
                </DropdownMenuContent>
              </DropdownMenu>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant={sidebarOpen ? "secondary" : "ghost"}
                      size="icon-sm"
                      className={sidebarOpen ? "" : "text-muted-foreground"}
                      onClick={handleToggleSidebar}
                    >
                      <PanelRight />
                    </Button>
                  }
                />
                <TooltipContent side="bottom">{t(($) => $.detail.sidebar_tooltip)}</TooltipContent>
              </Tooltip>
              </>
            }
          />

          <div className={cn("flex min-h-0 flex-1 flex-col overflow-y-auto pt-4", PAGE_GUTTER)}>
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="text-body font-medium">{t(($) => $.detail.section_projects)}</h2>
              <div className="flex items-center gap-1">
                <Popover
                  open={attachOpen}
                  onOpenChange={(v) => {
                    setAttachOpen(v);
                    if (!v) setAttachFilter("");
                  }}
                >
                  <PopoverTrigger
                    render={
                      <Button variant="outline" size="sm">
                        <FolderKanban className="size-3.5" />
                        {t(($) => $.detail.empty_projects_attach)}
                      </Button>
                    }
                  />
                  <PopoverContent align="end" className="w-64 p-0">
                    <div className="border-b px-2 py-1.5">
                      <div className="relative">
                        <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                        <Input
                          value={attachFilter}
                          onChange={(e) => setAttachFilter(e.target.value)}
                          placeholder={t(($) => $.detail.attach_search_placeholder)}
                          className="h-8 pl-7 text-body"
                        />
                      </div>
                    </div>
                    <div className="max-h-64 overflow-y-auto p-1">
                      {attachableProjects.length === 0 ? (
                        <div className="px-2 py-3 text-center text-caption text-muted-foreground">
                          {t(($) => $.detail.attach_empty)}
                        </div>
                      ) : (
                        attachableProjects.map((p) => (
                          <button
                            type="button"
                            key={p.id}
                            onClick={() => handleAttach(p.id)}
                            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-body hover:bg-accent"
                          >
                            <ProjectIcon project={p} size="sm" />
                            <span className="min-w-0 truncate">{p.title}</span>
                          </button>
                        ))
                      )}
                    </div>
                  </PopoverContent>
                </Popover>
                <Button size="sm" onClick={openCreateProject}>
                  <Plus className="size-3.5" />
                  {t(($) => $.detail.empty_projects_create)}
                </Button>
              </div>
            </div>

            {childProjects.length === 0 ? (
              <div className="flex flex-1 flex-col items-center justify-center py-16 text-muted-foreground">
                <FolderKanban className="mb-3 h-10 w-10 opacity-30" />
                <p className="text-body font-medium text-foreground">{t(($) => $.detail.empty_projects_title)}</p>
                <p className="mt-1 max-w-sm text-center text-caption">{t(($) => $.detail.empty_projects_hint)}</p>
              </div>
            ) : (
              <div className="space-y-0.5 pb-8">
                {childProjects.map((project) => (
                  <ChildProjectRow
                    key={project.id}
                    project={project}
                    href={wsPaths.projectDetail(project.id)}
                  />
                ))}
              </div>
            )}
          </div>
          </div>
        </ResizablePanel>
        {!isMobile && <ResizableHandle />}
        {!isMobile && (
        <ResizablePanel
          id="sidebar"
          {...rightSidebarPanelMotionProps}
          data-right-sidebar-motion={desktopSidebarMotionEnabled ? "enabled" : undefined}
          defaultSize={desktopSidebarOpen ? 320 : 0}
          minSize={260}
          maxSize={420}
          collapsible
          groupResizeBehavior="preserve-pixel-size"
          panelRef={sidebarRef}
          onResize={handleDesktopSidebarResize}
        >
          <AnimatedRightSidebar open={desktopSidebarVisualOpen} motionEnabled={desktopSidebarMotionEnabled}>
            {sidebarContent}
          </AnimatedRightSidebar>
        </ResizablePanel>
        )}
        {isMobile && (
          <Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
            <SheetContent side="right" showCloseButton={false} className="w-[320px] overflow-y-auto p-4">
              {sidebarContent}
            </SheetContent>
          </Sheet>
        )}
      </ResizablePanelGroup>

      {isWorkspaceAdmin && (
        <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{t(($) => $.delete_dialog.title)}</AlertDialogTitle>
              <AlertDialogDescription>
                {t(($) => $.delete_dialog.description)}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t(($) => $.delete_dialog.cancel)}</AlertDialogCancel>
              <AlertDialogAction onClick={handleDelete} className="bg-destructive text-white hover:bg-destructive/90">
                {t(($) => $.delete_dialog.confirm)}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </>
  );
}
