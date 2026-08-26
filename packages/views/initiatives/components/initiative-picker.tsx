"use client";

import { useState } from "react";
import { Target } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { initiativeListOptions } from "@multica/core/initiatives/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { InitiativeIcon } from "./initiative-icon";
import {
  PropertyPicker,
  PickerItem,
  PickerEmpty,
  PICKER_TRIGGER_CLASS,
} from "../../issues/components/pickers/property-picker";
import { matchesPinyin } from "../../editor/extensions/pinyin-match";
import { useT } from "../../i18n";

export function InitiativePicker({
  initiativeId,
  onUpdate,
  triggerRender,
  align = "start",
  defaultOpen = false,
  open: controlledOpen,
  onOpenChange,
  disabled = false,
}: {
  initiativeId: string | null;
  onUpdate: (updates: { initiative_id: string | null }) => void;
  triggerRender?: React.ReactElement;
  align?: "start" | "center" | "end";
  defaultOpen?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  disabled?: boolean;
}) {
  const { t } = useT("initiatives");
  const wsId = useWorkspaceId();
  const { data: initiatives = [] } = useQuery(initiativeListOptions(wsId));
  const current = initiatives.find((item) => item.id === initiativeId);
  const [filter, setFilter] = useState("");
  const [internalOpen, setInternalOpen] = useState(defaultOpen);
  const open = disabled ? false : controlledOpen ?? internalOpen;
  const setOpen = disabled ? () => {} : onOpenChange ?? setInternalOpen;

  const query = filter.trim().toLowerCase();
  const filtered = initiatives.filter(
    (item) => item.title.toLowerCase().includes(query) || matchesPinyin(item.title, query),
  );

  const resolvedTriggerRender = triggerRender ?? (
    <button type="button" disabled={disabled} className={PICKER_TRIGGER_CLASS} />
  );

  return (
    <div className="inline-flex min-w-0">
      <PropertyPicker
        open={open}
        onOpenChange={setOpen}
        width="w-52"
        align={align}
        searchable
        searchPlaceholder={t(($) => $.picker.search_placeholder)}
        onSearchChange={setFilter}
        triggerRender={resolvedTriggerRender}
        trigger={
          current ? (
            <>
              <InitiativeIcon initiative={current} size="sm" />
              <span className="truncate">{current.title}</span>
            </>
          ) : (
            <>
              <Target className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              <span className="truncate">{t(($) => $.picker.no_initiative)}</span>
            </>
          )
        }
      >
        <PickerItem
          emptyValue
          selected={!initiativeId}
          onClick={() => {
            onUpdate({ initiative_id: null });
            setOpen(false);
          }}
        >
          <Target className="h-3.5 w-3.5 text-muted-foreground" />
          <span className="text-muted-foreground">{t(($) => $.picker.no_initiative)}</span>
        </PickerItem>

        {filtered.map((item) => (
          <PickerItem
            key={item.id}
            selected={item.id === initiativeId}
            onClick={() => {
              onUpdate({ initiative_id: item.id });
              setOpen(false);
            }}
          >
            <InitiativeIcon initiative={item} size="sm" />
            <span className="truncate">{item.title}</span>
          </PickerItem>
        ))}

        {initiatives.length === 0 && (
          <div className="px-2 py-1.5 text-caption text-muted-foreground">{t(($) => $.picker.empty)}</div>
        )}
        {initiatives.length > 0 && filtered.length === 0 && query && <PickerEmpty />}
      </PropertyPicker>
    </div>
  );
}
