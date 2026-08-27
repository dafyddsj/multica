"use client";

import { Check } from "lucide-react";
import {
  INITIATIVE_PRIORITY_CONFIG,
  INITIATIVE_PRIORITY_ORDER,
} from "@multica/core/initiatives/config";
import { cn } from "@multica/ui/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import type {
  Initiative,
  InitiativeStatus,
  InitiativePriority,
  UpdateInitiativeRequest,
} from "@multica/core/types";
import { PriorityIcon } from "../../issues/components/priority-icon";
import { useEntityStatusPicker } from "../../common/entity-status-picker";
import { useInitiativePriorityLabels } from "./labels";

export function InitiativeStatusBadge({
  initiative,
  handleUpdate,
  triggerClassName,
  align = "end",
}: {
  initiative: Initiative;
  handleUpdate: (data: UpdateInitiativeRequest) => void;
  triggerClassName?: string;
  align?: "start" | "end" | "center";
}) {
  const { options, current } = useEntityStatusPicker("initiative");
  const selected = current(initiative.status);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            className={cn(
              "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-caption font-medium cursor-pointer hover:opacity-80 transition-opacity",
              selected.badgeBg,
              selected.badgeText,
              triggerClassName,
            )}
          >
            {selected.label}
          </button>
        }
      />
      <DropdownMenuContent align={align} className="w-44">
        {options.map((s) => (
          <DropdownMenuItem
            key={s.key}
            onClick={() => handleUpdate({ status: s.key as InitiativeStatus })}
          >
            <span
              className={cn("size-2 rounded-full", !s.hex && s.dotClass)}
              style={s.hex ? { backgroundColor: s.hex } : undefined}
            />
            <span>{s.label}</span>
            {s.key === initiative.status && <Check className="ml-auto h-3.5 w-3.5" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function InitiativePriorityBadge({
  initiative,
  handleUpdate,
  triggerClassName,
  align = "end",
}: {
  initiative: Initiative;
  handleUpdate: (data: UpdateInitiativeRequest) => void;
  triggerClassName?: string;
  align?: "start" | "end" | "center";
}) {
  const priorityLabels = useInitiativePriorityLabels();
  const priorityCfg = INITIATIVE_PRIORITY_CONFIG[initiative.priority];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            className={cn(
              "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-caption font-medium hover:bg-accent/60 transition-colors cursor-pointer",
              triggerClassName,
            )}
          >
            <PriorityIcon priority={initiative.priority} />
            <span className={cn("text-caption", priorityCfg.color)}>
              {priorityLabels[initiative.priority]}
            </span>
          </button>
        }
      />
      <DropdownMenuContent align={align} className="w-44">
        {INITIATIVE_PRIORITY_ORDER.map((p) => (
          <DropdownMenuItem
            key={p}
            onClick={() => handleUpdate({ priority: p as InitiativePriority })}
          >
            <PriorityIcon priority={p} />
            <span>{priorityLabels[p]}</span>
            {p === initiative.priority && <Check className="ml-auto h-3.5 w-3.5" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
