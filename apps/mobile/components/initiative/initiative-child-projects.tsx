import { useMemo } from "react";
import { View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { router } from "expo-router";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { ProjectRow } from "@/components/project/project-row";
import { projectListOptions } from "@/data/queries/projects";
import { projectsForInitiative } from "@/data/queries/initiatives";
import { useWorkspaceStore } from "@/data/workspace-store";

interface Props {
  initiativeId: string;
  onAdd?: () => void;
}

export function InitiativeChildProjects({ initiativeId, onAdd }: Props) {
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const wsSlug = useWorkspaceStore((s) => s.currentWorkspaceSlug);
  const { data, isLoading, error, refetch } = useQuery(projectListOptions(wsId));

  const children = useMemo(() => {
    const list = projectsForInitiative(data ?? [], initiativeId);
    return [...list].sort(
      (a, b) =>
        new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
    );
  }, [data, initiativeId]);

  if (isLoading) {
    return (
      <View className="px-4 py-6">
        <Text className="text-sm text-muted-foreground">Loading projects…</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View className="px-4 py-6 gap-3">
        <Text className="text-sm text-destructive">
          Failed to load projects:{" "}
          {error instanceof Error ? error.message : "unknown error"}
        </Text>
        <Button variant="outline" onPress={() => refetch()}>
          <Text>Retry</Text>
        </Button>
      </View>
    );
  }

  return (
    <View>
      <View className="flex-row items-center justify-between px-4 py-2">
        <Text className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
          Projects
        </Text>
        <Text className="text-xs text-muted-foreground/60">
          {children.length}
        </Text>
      </View>
      {children.length === 0 ? (
        <View className="px-4 py-6 gap-3">
          <Text className="text-sm text-muted-foreground">
            No projects yet. Issues attach through projects, not directly to
            this initiative.
          </Text>
          {onAdd ? (
            <Button variant="outline" onPress={onAdd}>
              <Text>Add project</Text>
            </Button>
          ) : null}
        </View>
      ) : (
        children.map((project) => (
          <ProjectRow
            key={project.id}
            project={project}
            onPress={() => {
              if (wsSlug) router.push(`/${wsSlug}/project/${project.id}`);
            }}
          />
        ))
      )}
    </View>
  );
}
