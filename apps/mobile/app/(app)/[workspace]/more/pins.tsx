import { useMemo } from "react";
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  View,
} from "react-native";
import { useQuery } from "@tanstack/react-query";
import { router } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import type { Initiative, Issue, PinnedItem, Project } from "@multica/core/types";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { IssueRow } from "@/components/issue/issue-row";
import { InitiativeRow } from "@/components/initiative/initiative-row";
import { ProjectRow } from "@/components/project/project-row";
import { pinListOptions } from "@/data/queries/pins";
import { useDeletePin } from "@/data/mutations/pins";
import { issueDetailOptions } from "@/data/queries/issues";
import { initiativeDetailOptions } from "@/data/queries/initiatives";
import { projectDetailOptions } from "@/data/queries/projects";
import { useAuthStore } from "@/data/auth-store";
import { useWorkspaceStore } from "@/data/workspace-store";
import { useColorScheme } from "@/lib/use-color-scheme";
import { THEME } from "@/lib/theme";

export default function PinsPage() {
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const wsSlug = useWorkspaceStore((s) => s.currentWorkspaceSlug);
  const userId = useAuthStore((s) => s.user?.id ?? null);

  const { data, isLoading, error, refetch, isRefetching } = useQuery(
    pinListOptions(wsId, userId),
  );

  // Sort by `position` ascending so the order matches web's sidebar
  // (the reorder endpoint writes 1-based positions there too).
  const pins = useMemo(
    () =>
      [...(data ?? [])]
        .filter(
          (p) =>
            p.item_type === "issue" ||
            p.item_type === "project" ||
            p.item_type === "initiative",
        )
        .sort((a, b) => a.position - b.position),
    [data],
  );

  if (isLoading) {
    return (
      <View className="flex-1 items-center justify-center bg-background">
        <ActivityIndicator />
      </View>
    );
  }

  if (error) {
    return (
      <View className="flex-1 bg-background px-4 gap-3 pt-4">
        <Text className="text-sm text-destructive">
          Failed to load pins:{" "}
          {error instanceof Error ? error.message : "unknown error"}
        </Text>
        <Button variant="outline" onPress={() => refetch()}>
          <Text>Retry</Text>
        </Button>
      </View>
    );
  }

  if (pins.length === 0) {
    return (
      <View className="flex-1 items-center justify-center bg-background px-6">
        <Text className="text-sm text-muted-foreground text-center">
          No pins yet. Pin an issue, project, or initiative from its actions
          menu to surface it here.
        </Text>
      </View>
    );
  }

  return (
    <ScrollView
      className="flex-1 bg-background"
      contentContainerClassName="pb-6"
      refreshControl={
        <RefreshControl
          refreshing={isRefetching}
          onRefresh={() => refetch()}
        />
      }
      showsVerticalScrollIndicator={false}
    >
      {pins.map((pin, idx) => (
        <View key={pin.id}>
          {idx > 0 ? <View className="h-px bg-border ml-4" /> : null}
          <PinRow pin={pin} wsId={wsId} wsSlug={wsSlug} />
        </View>
      ))}
    </ScrollView>
  );
}

function PinRow({
  pin,
  wsId,
  wsSlug,
}: {
  pin: PinnedItem;
  wsId: string | null;
  wsSlug: string | null;
}) {
  if (pin.item_type === "issue") {
    return <IssuePinRow pin={pin} wsId={wsId} wsSlug={wsSlug} />;
  }
  if (pin.item_type === "project") {
    return <ProjectPinRow pin={pin} wsId={wsId} wsSlug={wsSlug} />;
  }
  if (pin.item_type === "initiative") {
    return <InitiativePinRow pin={pin} wsId={wsId} wsSlug={wsSlug} />;
  }
  return null;
}

function IssuePinRow({
  pin,
  wsId,
  wsSlug,
}: {
  pin: PinnedItem;
  wsId: string | null;
  wsSlug: string | null;
}) {
  const { data, isLoading } = useQuery(issueDetailOptions(wsId, pin.item_id));
  // EMPTY_ISSUE_FALLBACK has an empty id — treat as deleted/no-access.
  const issue = data && data.id ? (data as Issue) : null;

  if (isLoading) return <SkeletonRow />;
  if (!issue)
    return <MissingPinRow itemType="issue" itemId={pin.item_id} />;

  return (
    <IssueRow
      issue={issue}
      showStatus
      onPress={() => {
        if (wsSlug) router.push(`/${wsSlug}/issue/${issue.id}`);
      }}
    />
  );
}

function ProjectPinRow({
  pin,
  wsId,
  wsSlug,
}: {
  pin: PinnedItem;
  wsId: string | null;
  wsSlug: string | null;
}) {
  const { data, isLoading } = useQuery(
    projectDetailOptions(wsId, pin.item_id),
  );
  const project = data && data.id ? (data as Project) : null;

  if (isLoading) return <SkeletonRow />;
  if (!project)
    return <MissingPinRow itemType="project" itemId={pin.item_id} />;

  return (
    <ProjectRow
      project={project}
      onPress={() => {
        if (wsSlug) router.push(`/${wsSlug}/project/${project.id}`);
      }}
    />
  );
}

function InitiativePinRow({
  pin,
  wsId,
  wsSlug,
}: {
  pin: PinnedItem;
  wsId: string | null;
  wsSlug: string | null;
}) {
  const { data, isLoading } = useQuery(
    initiativeDetailOptions(wsId, pin.item_id),
  );
  const initiative = data && data.id ? (data as Initiative) : null;

  if (isLoading) return <SkeletonRow />;
  if (!initiative)
    return <MissingPinRow itemType="initiative" itemId={pin.item_id} />;

  return (
    <InitiativeRow
      initiative={initiative}
      onPress={() => {
        if (wsSlug) router.push(`/${wsSlug}/initiative/${initiative.id}`);
      }}
    />
  );
}

function SkeletonRow() {
  return (
    <View className="px-4 py-3 flex-row items-center gap-3">
      <View className="size-5 rounded bg-muted" />
      <View className="flex-1 h-4 rounded bg-muted" />
    </View>
  );
}

/**
 * Renders for pins whose target issue/project was deleted or revoked.
 * Tapping triggers unpin so the user can clean it up; no destination
 * navigation since there's nothing to navigate to. Subtle styling so
 * it doesn't dominate the list of live pins.
 */
function MissingPinRow({
  itemType,
  itemId,
}: {
  itemType: "issue" | "project" | "initiative";
  itemId: string;
}) {
  const { colorScheme } = useColorScheme();
  const deletePin = useDeletePin();
  return (
    <Pressable
      onPress={() => deletePin.mutate({ itemType, itemId })}
      className="px-4 py-3 flex-row items-center gap-3 active:bg-secondary opacity-60"
      accessibilityLabel={`Unavailable ${itemType}, tap to unpin`}
    >
      <Ionicons
        name="alert-circle-outline"
        size={18}
        color={THEME[colorScheme].mutedForeground}
      />
      <Text className="flex-1 text-sm text-muted-foreground" numberOfLines={1}>
        Unavailable {itemType} — tap to unpin
      </Text>
    </Pressable>
  );
}
