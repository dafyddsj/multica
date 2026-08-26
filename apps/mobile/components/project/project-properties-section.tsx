/**
 * Project properties section. Tappable rows for Status / Priority / Lead.
 * Each row opens a picker sheet via the corresponding `onPress*` callback.
 *
 * Layout mirrors iOS Settings rows: label on left, current value on right
 * with a disclosure chevron, full-width separator below each row. Tapping
 * anywhere on the row triggers the picker.
 *
 * Lead supports both member and agent (Project.lead_type), resolved via
 * useActorLookup so it shares the same lookup with my-issues + issue detail.
 */
import { Pressable, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { Ionicons } from "@expo/vector-icons";
import type { Project } from "@multica/core/types";
import { Text } from "@/components/ui/text";
import { ActorAvatar } from "@/components/ui/actor-avatar";
import { InitiativeIcon } from "@/components/ui/initiative-icon";
import { ProjectStatusIcon } from "@/components/ui/project-status-icon";
import { ProjectPriorityIcon } from "@/components/ui/project-priority-icon";
import {
  projectPriorityLabel,
  projectStatusLabel,
} from "@/lib/project-status";
import { useActorLookup } from "@/data/use-actor-name";
import {
  findInitiative,
  initiativeListOptions,
} from "@/data/queries/initiatives";
import { useWorkspaceStore } from "@/data/workspace-store";
import { useColorScheme } from "@/lib/use-color-scheme";
import { THEME } from "@/lib/theme";

interface Props {
  project: Project;
  onPressStatus: () => void;
  onPressPriority: () => void;
  onPressLead: () => void;
  onPressInitiative: () => void;
}

export function ProjectPropertiesSection({
  project,
  onPressStatus,
  onPressPriority,
  onPressLead,
  onPressInitiative,
}: Props) {
  const { getName } = useActorLookup();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const { data: initiatives = [] } = useQuery(initiativeListOptions(wsId));
  const initiative = findInitiative(initiatives, project.initiative_id);
  const leadName =
    project.lead_type && project.lead_id
      ? getName(project.lead_type, project.lead_id)
      : null;

  return (
    <View className="border-y border-border bg-background">
      <Row
        label="Status"
        onPress={onPressStatus}
        left={<ProjectStatusIcon status={project.status} size={16} />}
        right={
          <Text className="text-sm text-foreground">
            {projectStatusLabel(project.status)}
          </Text>
        }
      />
      <Separator />
      <Row
        label="Priority"
        onPress={onPressPriority}
        left={<ProjectPriorityIcon priority={project.priority} size={16} />}
        right={
          <Text className="text-sm text-foreground">
            {projectPriorityLabel(project.priority)}
          </Text>
        }
      />
      <Separator />
      <Row
        label="Lead"
        onPress={onPressLead}
        left={
          leadName ? (
            <ActorAvatar
              type={project.lead_type}
              id={project.lead_id}
              size={20}
              showPresence
            />
          ) : (
            <PlaceholderAvatar />
          )
        }
        right={
          <Text
            className={
              leadName
                ? "text-sm text-foreground"
                : "text-sm text-muted-foreground"
            }
          >
            {leadName ?? "Unassigned"}
          </Text>
        }
      />
      <Separator />
      <Row
        label="Initiative"
        onPress={onPressInitiative}
        left={
          initiative ? (
            <InitiativeIcon icon={initiative.icon} size="sm" />
          ) : (
            <View style={{ width: 18, height: 18 }} />
          )
        }
        right={
          <Text
            className={
              initiative
                ? "text-sm text-foreground"
                : "text-sm text-muted-foreground"
            }
          >
            {initiative?.title ?? "None"}
          </Text>
        }
      />
    </View>
  );
}

function Row({
  label,
  onPress,
  left,
  right,
}: {
  label: string;
  onPress: () => void;
  left: React.ReactNode;
  right: React.ReactNode;
}) {
  return (
    <Pressable
      onPress={onPress}
      className="flex-row items-center gap-3 px-4 py-3 active:bg-secondary"
    >
      <Text className="text-sm text-muted-foreground w-24">{label}</Text>
      <View className="flex-row items-center gap-2 flex-1">
        {left}
        {right}
      </View>
      <Chevron />
    </Pressable>
  );
}

function Separator() {
  return <View className="h-px bg-border ml-4" />;
}

function Chevron() {
  const { colorScheme } = useColorScheme();
  return (
    <Ionicons
      name="chevron-forward"
      size={14}
      color={THEME[colorScheme].mutedForeground}
    />
  );
}

function PlaceholderAvatar() {
  return (
    <View
      style={{ width: 20, height: 20, borderRadius: 10 }}
      className="border border-dashed border-muted-foreground/40"
    />
  );
}
