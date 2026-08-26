/**
 * Pure picker body for a project's parent initiative. Clone of
 * `ProjectPickerBody` with a "No initiative" row.
 */
import { useMemo } from "react";
import { FlatList, Pressable, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { Ionicons } from "@expo/vector-icons";
import { useColorScheme } from "nativewind";
import type { Initiative } from "@multica/core/types";
import { Text } from "@/components/ui/text";
import { InitiativeIcon } from "@/components/ui/initiative-icon";
import { MOBILE_PLACEHOLDER_COLOR } from "@/components/ui/input-tokens";
import { initiativeListOptions } from "@/data/queries/initiatives";
import { useWorkspaceStore } from "@/data/workspace-store";
import { useScrollToTopOnChange } from "@/lib/use-scroll-to-top-on-change";
import { THEME } from "@/lib/theme";

type Row = { kind: "none" } | { kind: "initiative"; initiative: Initiative };

interface Props {
  value: Initiative | null;
  query: string;
  onChange: (next: Initiative | null) => void;
}

export function InitiativePickerBody({ value, query, onChange }: Props) {
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const { data: initiatives = [] } = useQuery(initiativeListOptions(wsId));
  const listRef = useScrollToTopOnChange(query);
  const { colorScheme } = useColorScheme();
  const checkColor =
    colorScheme === "dark" ? THEME.dark.primary : THEME.light.primary;

  const rows = useMemo<Row[]>(() => {
    const q = query.trim().toLowerCase();
    const matchName = (n: string) => !q || n.toLowerCase().includes(q);
    const initiativeRows: Row[] = [...initiatives]
      .filter((item) => matchName(item.title))
      .sort((a, b) => a.title.localeCompare(b.title))
      .map((item) => ({ kind: "initiative" as const, initiative: item }));

    if (q) return initiativeRows;

    const selected = initiativeRows.find(
      (r) => r.kind === "initiative" && r.initiative.id === value?.id,
    );
    return [
      { kind: "none" },
      ...(selected ? [selected] : []),
      ...initiativeRows.filter(
        (r) => !(r.kind === "initiative" && r.initiative.id === value?.id),
      ),
    ];
  }, [initiatives, query, value]);

  const isSelected = (row: Row) => {
    if (row.kind === "none") return value === null;
    return value !== null && row.initiative.id === value.id;
  };

  return (
    <FlatList
      ref={listRef}
      data={rows}
      className="flex-1"
      keyboardShouldPersistTaps="handled"
      automaticallyAdjustKeyboardInsets
      contentInsetAdjustmentBehavior="automatic"
      keyExtractor={(row) =>
        row.kind === "none" ? "none" : `i:${row.initiative.id}`
      }
      renderItem={({ item }) => (
        <Pressable
          onPress={() =>
            item.kind === "none" ? onChange(null) : onChange(item.initiative)
          }
          className="flex-row items-center gap-3 px-4 py-3 active:bg-secondary"
        >
          {item.kind === "none" ? (
            <Ionicons
              name="close-circle-outline"
              size={28}
              color={MOBILE_PLACEHOLDER_COLOR}
            />
          ) : (
            <InitiativeIcon icon={item.initiative.icon} size="md" />
          )}
          <Text
            className="flex-1 text-base text-foreground"
            numberOfLines={1}
          >
            {item.kind === "none" ? "No initiative" : item.initiative.title}
          </Text>
          {isSelected(item) ? (
            <Ionicons name="checkmark" size={20} color={checkColor} />
          ) : null}
        </Pressable>
      )}
      ListEmptyComponent={
        <View className="px-3 py-8 items-center">
          <Text className="text-sm text-muted-foreground text-center">
            {query
              ? "No matches."
              : "No initiatives in this workspace yet."}
          </Text>
        </View>
      }
    />
  );
}
