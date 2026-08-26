import { useCallback, useState } from "react";
import {
  Alert,
  InteractionManager,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  TextInput,
  View,
} from "react-native";
import { Stack, router } from "expo-router";
import { Text } from "@/components/ui/text";
import { AutosizeTextArea } from "@/components/ui/autosize-textarea";
import {
  MIN_BODY_INPUT_HEIGHT_PX,
  MOBILE_PLACEHOLDER_COLOR,
} from "@/components/ui/input-tokens";
import { ProjectStatusIcon } from "@/components/ui/project-status-icon";
import { ProjectPriorityIcon } from "@/components/ui/project-priority-icon";
import {
  projectPriorityLabel,
  projectStatusLabel,
} from "@/lib/project-status";
import { useCreateInitiative } from "@/data/mutations/initiatives";
import { useNewInitiativeDraftStore } from "@/data/stores/new-initiative-draft-store";
import { useWorkspaceStore } from "@/data/workspace-store";

type NewInitiativePickerField = "status" | "priority";
const NEW_INITIATIVE_PICKER_PATHNAMES = {
  status: "/[workspace]/new-initiative-picker/status",
  priority: "/[workspace]/new-initiative-picker/priority",
} as const satisfies Record<NewInitiativePickerField, string>;

export default function NewInitiative() {
  const wsSlug = useWorkspaceStore((s) => s.currentWorkspaceSlug);
  const create = useCreateInitiative();

  const [title, setTitle] = useState("");
  const [icon, setIcon] = useState("");
  const [description, setDescription] = useState("");
  const status = useNewInitiativeDraftStore((s) => s.status);
  const priority = useNewInitiativeDraftStore((s) => s.priority);
  const resetDraft = useNewInitiativeDraftStore((s) => s.reset);

  const dirty =
    title.length > 0 ||
    icon.length > 0 ||
    description.length > 0 ||
    status !== "planned" ||
    priority !== "none";

  const canCreate = title.trim().length > 0 && !create.isPending;

  const openPicker = useCallback(
    (field: NewInitiativePickerField) => {
      if (!wsSlug) return;
      router.push({
        pathname: NEW_INITIATIVE_PICKER_PATHNAMES[field],
        params: { workspace: wsSlug },
      });
    },
    [wsSlug],
  );

  const onCancel = useCallback(() => {
    if (!dirty) {
      resetDraft();
      router.back();
      return;
    }
    Alert.alert("Discard initiative?", "Your draft will be lost.", [
      { text: "Keep editing", style: "cancel" },
      {
        text: "Discard",
        style: "destructive",
        onPress: () => {
          resetDraft();
          router.back();
        },
      },
    ]);
  }, [dirty, resetDraft]);

  const onCreate = useCallback(() => {
    if (!canCreate) return;
    create.mutate(
      {
        title: title.trim(),
        description: description.trim() || undefined,
        icon: icon.trim() || undefined,
        status,
        priority,
      },
      {
        onSuccess: (initiative) => {
          resetDraft();
          router.back();
          InteractionManager.runAfterInteractions(() => {
            if (wsSlug) router.push(`/${wsSlug}/initiative/${initiative.id}`);
          });
        },
        onError: (err) => {
          Alert.alert(
            "Failed to create initiative",
            err instanceof Error ? err.message : "Unknown error",
          );
        },
      },
    );
  }, [
    canCreate,
    create,
    title,
    description,
    icon,
    status,
    priority,
    wsSlug,
    resetDraft,
  ]);

  const headerLeft = useCallback(() => {
    return (
      <Pressable onPress={onCancel} className="px-1 py-1">
        <Text className="text-base text-brand">Cancel</Text>
      </Pressable>
    );
  }, [onCancel]);

  const headerRight = useCallback(() => {
    return (
      <Pressable
        onPress={onCreate}
        disabled={!canCreate}
        className={canCreate ? "px-1 py-1" : "px-1 py-1 opacity-40"}
      >
        <Text className="text-base text-brand font-semibold">
          {create.isPending ? "Creating…" : "Create"}
        </Text>
      </Pressable>
    );
  }, [canCreate, onCreate, create.isPending]);

  return (
    <>
      <Stack.Screen options={{ headerLeft, headerRight }} />
      <KeyboardAvoidingView
        className="flex-1 bg-background"
        behavior={Platform.OS === "ios" ? "padding" : undefined}
      >
        <ScrollView
          className="flex-1"
          contentContainerClassName="px-4 pt-4 pb-6 gap-4"
          keyboardShouldPersistTaps="handled"
        >
          <Field label="Icon (emoji)">
            <TextInput
              value={icon}
              onChangeText={(v) => setIcon(v.slice(0, 4))}
              placeholder="🎯"
              placeholderTextColor={MOBILE_PLACEHOLDER_COLOR}
              className="text-2xl text-foreground bg-secondary/50 rounded-md px-3 py-2 self-start min-w-[60px] text-center"
              maxLength={4}
            />
          </Field>

          <Field label="Title">
            <TextInput
              value={title}
              onChangeText={setTitle}
              placeholder="Initiative title"
              placeholderTextColor={MOBILE_PLACEHOLDER_COLOR}
              className="text-base text-foreground bg-secondary/50 rounded-md px-3 py-2"
              autoFocus
              returnKeyType="next"
            />
          </Field>

          <Field label="Description">
            <AutosizeTextArea
              value={description}
              onChangeText={setDescription}
              placeholder="What is this initiative about?"
              className="bg-secondary/50 rounded-md px-3 py-2"
              minHeight={MIN_BODY_INPUT_HEIGHT_PX}
            />
          </Field>

          <View className="flex-row gap-2">
            <View className="flex-1">
              <Field label="Status">
                <Pressable
                  onPress={() => openPicker("status")}
                  className="flex-row items-center gap-2 bg-secondary/50 rounded-md px-3 py-2.5"
                >
                  <ProjectStatusIcon status={status} size={16} />
                  <Text className="text-sm text-foreground flex-1">
                    {projectStatusLabel(status)}
                  </Text>
                </Pressable>
              </Field>
            </View>
            <View className="flex-1">
              <Field label="Priority">
                <Pressable
                  onPress={() => openPicker("priority")}
                  className="flex-row items-center gap-2 bg-secondary/50 rounded-md px-3 py-2.5"
                >
                  <ProjectPriorityIcon priority={priority} size={16} />
                  <Text className="text-sm text-foreground flex-1">
                    {projectPriorityLabel(priority)}
                  </Text>
                </Pressable>
              </Field>
            </View>
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <View className="gap-1.5">
      <Text className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </Text>
      {children}
    </View>
  );
}
