import { Pressable, View } from "react-native";
import type { Initiative } from "@multica/core/types";
import { Text } from "@/components/ui/text";
import { InitiativeIcon } from "@/components/ui/initiative-icon";
import { ProjectStatusIcon } from "@/components/ui/project-status-icon";
import { ProjectPriorityIcon } from "@/components/ui/project-priority-icon";
import {
  projectPriorityLabel,
  projectStatusLabel,
} from "@/lib/project-status";
import { timeAgo } from "@/lib/time-ago";

interface Props {
  initiative: Initiative;
  onPress: () => void;
}

export function InitiativeRow({ initiative, onPress }: Props) {
  const totalIssues = initiative.issue_count;
  const showCount = totalIssues > 0;

  return (
    <Pressable onPress={onPress} className="active:bg-secondary px-4 py-3">
      <View className="flex-row items-start gap-3">
        <InitiativeIcon icon={initiative.icon} size="lg" />
        <View className="flex-1 gap-1">
          <Text
            className="text-base text-foreground font-medium"
            numberOfLines={1}
          >
            {initiative.title}
          </Text>
          <View className="flex-row items-center gap-3">
            <View className="flex-row items-center gap-1.5">
              <ProjectStatusIcon status={initiative.status} size={12} />
              <Text className="text-xs text-muted-foreground">
                {projectStatusLabel(initiative.status)}
              </Text>
            </View>
            {initiative.priority !== "none" ? (
              <View className="flex-row items-center gap-1.5">
                <ProjectPriorityIcon priority={initiative.priority} size={12} />
                <Text className="text-xs text-muted-foreground">
                  {projectPriorityLabel(initiative.priority)}
                </Text>
              </View>
            ) : null}
          </View>
        </View>
        <View className="items-end gap-1">
          {showCount ? (
            <Text className="text-xs text-muted-foreground tabular-nums">
              {initiative.done_count}/{totalIssues}
            </Text>
          ) : (
            <Text className="text-xs text-muted-foreground/60">
              {initiative.project_count > 0
                ? `${initiative.project_count}`
                : "—"}
            </Text>
          )}
          <Text className="text-[11px] text-muted-foreground/70">
            {timeAgo(initiative.updated_at)}
          </Text>
        </View>
      </View>
    </Pressable>
  );
}
