/**
 * Initiative detail. Properties + child projects. No board, gantt, or
 * IssueSurface — issues attach only through projects.
 */
import { useCallback } from "react";
import {
  ActionSheetIOS,
  ActivityIndicator,
  Alert,
  Linking,
  RefreshControl,
  ScrollView,
  View,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { Stack, router, useLocalSearchParams } from "expo-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { InitiativeHeaderCard } from "@/components/initiative/initiative-header-card";
import { InitiativePropertiesSection } from "@/components/initiative/initiative-properties-section";
import { InitiativeChildProjects } from "@/components/initiative/initiative-child-projects";
import { initiativeDetailOptions } from "@/data/queries/initiatives";
import { projectKeys } from "@/data/queries/projects";
import { pinListOptions } from "@/data/queries/pins";
import { useDeleteInitiative } from "@/data/mutations/initiatives";
import { useCreatePin, useDeletePin } from "@/data/mutations/pins";
import { useAuthStore } from "@/data/auth-store";
import { useInitiativeRealtime } from "@/data/realtime/use-initiative-realtime";
import { useNewProjectDraftStore } from "@/data/stores/new-project-draft-store";
import { useWorkspaceStore } from "@/data/workspace-store";

export default function InitiativeDetail() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const wsSlug = useWorkspaceStore((s) => s.currentWorkspaceSlug);
  const qc = useQueryClient();

  const detail = useQuery(initiativeDetailOptions(wsId, id));
  const deleteInitiative = useDeleteInitiative(id);

  useInitiativeRealtime(id, () => router.back());

  const onRefresh = useCallback(async () => {
    await Promise.all([
      detail.refetch(),
      qc.invalidateQueries({ queryKey: projectKeys.list(wsId) }),
    ]);
  }, [detail, qc, wsId]);

  const initiative = detail.data;
  const initiativeMissing = !initiative || initiative.id === "";

  const userId = useAuthStore((s) => s.user?.id ?? null);
  const { data: pins } = useQuery(pinListOptions(wsId, userId));
  const isPinned =
    !!initiative &&
    !!pins?.some(
      (p) => p.item_type === "initiative" && p.item_id === initiative.id,
    );
  const createPin = useCreatePin();
  const deletePin = useDeletePin();

  const onPressMore = () => {
    if (!initiative) return;
    const wsUrl = process.env.EXPO_PUBLIC_WEB_URL;
    const options = [
      "Cancel",
      isPinned ? "Unpin" : "Pin",
      "Edit details",
      ...(wsUrl ? ["Open on web"] : []),
      "Delete",
    ];
    const destructiveIndex = options.length - 1;
    ActionSheetIOS.showActionSheetWithOptions(
      {
        options,
        cancelButtonIndex: 0,
        destructiveButtonIndex: destructiveIndex,
      },
      (i) => {
        const label = options[i];
        if (label === "Pin") {
          createPin.mutate({
            item_type: "initiative",
            item_id: initiative.id,
          });
          return;
        }
        if (label === "Unpin") {
          deletePin.mutate({
            itemType: "initiative",
            itemId: initiative.id,
          });
          return;
        }
        if (label === "Edit details") {
          if (wsSlug) router.push(`/${wsSlug}/initiative/${id}/edit`);
          return;
        }
        if (label === "Open on web" && wsUrl) {
          Linking.openURL(`${wsUrl}/${wsSlug}/initiatives/${id}`);
          return;
        }
        if (i === destructiveIndex) {
          onDelete();
        }
      },
    );
  };

  const onDelete = () => {
    Alert.alert(
      "Delete initiative?",
      "Projects stay in the workspace and are detached from this initiative. This cannot be undone.",
      [
        { text: "Cancel", style: "cancel" },
        {
          text: "Delete",
          style: "destructive",
          onPress: () => {
            deleteInitiative.mutate(undefined, {
              onSuccess: () => router.back(),
            });
          },
        },
      ],
    );
  };

  const onAddProject = () => {
    if (!wsSlug || !initiative) return;
    useNewProjectDraftStore.getState().setInitiative(initiative);
    router.push(`/${wsSlug}/project/new`);
  };

  return (
    <SafeAreaView className="flex-1 bg-background" edges={["bottom"]}>
      <Stack.Screen
        options={{
          title: initiative?.title || "Initiative",
          headerBackTitle: "Back",
          headerRight: initiative
            ? () => (
                <IconButton
                  name="ellipsis-horizontal"
                  onPress={onPressMore}
                  accessibilityLabel="Initiative actions"
                />
              )
            : undefined,
        }}
      />
      {detail.isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator />
        </View>
      ) : detail.error || initiativeMissing ? (
        <View className="flex-1 items-center justify-center px-6 gap-3">
          <Text className="text-sm text-destructive text-center">
            Failed to load initiative:{" "}
            {detail.error instanceof Error
              ? detail.error.message
              : "not found"}
          </Text>
          <Button variant="outline" onPress={() => detail.refetch()}>
            <Text>Retry</Text>
          </Button>
        </View>
      ) : (
        <ScrollView
          contentContainerClassName="pb-10"
          refreshControl={
            <RefreshControl
              refreshing={detail.isRefetching}
              onRefresh={onRefresh}
            />
          }
          keyboardDismissMode="on-drag"
        >
          <InitiativeHeaderCard
            initiative={initiative}
            onEdit={() => {
              if (wsSlug) router.push(`/${wsSlug}/initiative/${id}/edit`);
            }}
          />
          <InitiativePropertiesSection
            initiative={initiative}
            onPressStatus={() => {
              if (wsSlug)
                router.push({
                  pathname: "/[workspace]/initiative/[id]/picker/status",
                  params: { workspace: wsSlug, id },
                });
            }}
            onPressPriority={() => {
              if (wsSlug)
                router.push({
                  pathname: "/[workspace]/initiative/[id]/picker/priority",
                  params: { workspace: wsSlug, id },
                });
            }}
            onPressLead={() => {
              if (wsSlug)
                router.push({
                  pathname: "/[workspace]/initiative/[id]/picker/lead",
                  params: { workspace: wsSlug, id },
                });
            }}
          />
          <View className="h-3" />
          <InitiativeChildProjects initiativeId={id} onAdd={onAddProject} />
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
