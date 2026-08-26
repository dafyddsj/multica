/**
 * Header card for initiative detail. Clone of `ProjectHeaderCard`, plus a
 * project_count line because the children of an initiative are projects,
 * not issues. Issue progress still uses done_count / issue_count.
 */
import { Pressable, View } from "react-native";
import type { Initiative } from "@multica/core/types";
import { Text } from "@/components/ui/text";
import { InitiativeIcon } from "@/components/ui/initiative-icon";

interface Props {
  initiative: Initiative;
  onEdit?: () => void;
}

export function InitiativeHeaderCard({ initiative, onEdit }: Props) {
  return (
    <Pressable
      onPress={onEdit}
      disabled={!onEdit}
      className="px-4 pt-4 pb-3 active:bg-secondary/40"
    >
      <View className="items-start gap-2">
        <InitiativeIcon icon={initiative.icon} size="lg" />
        <Text className="text-2xl font-bold text-foreground" selectable>
          {initiative.title}
        </Text>
        {initiative.description ? (
          <Text className="text-sm text-muted-foreground" selectable>
            {initiative.description}
          </Text>
        ) : onEdit ? (
          <Text className="text-sm text-muted-foreground/60 italic">
            Add a description
          </Text>
        ) : null}
        <Text className="text-xs text-muted-foreground">
          {initiative.project_count === 1
            ? "1 project"
            : `${initiative.project_count} projects`}
        </Text>
        {initiative.issue_count > 0 ? (
          <ProgressSection
            done={initiative.done_count}
            total={initiative.issue_count}
          />
        ) : null}
      </View>
    </Pressable>
  );
}

function ProgressSection({ done, total }: { done: number; total: number }) {
  const pct = Math.round((done / total) * 100);
  return (
    <View className="w-full pt-2 gap-1.5">
      <View className="flex-row items-center justify-between">
        <Text className="text-xs uppercase tracking-wider text-muted-foreground">
          Progress
        </Text>
        <Text className="text-xs text-muted-foreground">
          {done} / {total} · {pct}%
        </Text>
      </View>
      <View className="h-1.5 bg-secondary rounded-full overflow-hidden">
        <View
          className="h-full bg-brand rounded-full"
          style={{ width: `${pct}%` }}
        />
      </View>
    </View>
  );
}
