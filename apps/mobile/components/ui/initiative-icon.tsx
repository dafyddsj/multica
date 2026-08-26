/**
 * Initiative emoji icon. Clone of `ProjectIcon` with a 🎯 fallback so list
 * rows stay distinguishable from projects (📁).
 */
import { View } from "react-native";
import { Text } from "@/components/ui/text";

export type InitiativeIconSize = "sm" | "md" | "lg";

const SIZE: Record<InitiativeIconSize, { box: number; font: number }> = {
  sm: { box: 18, font: 14 },
  md: { box: 22, font: 16 },
  lg: { box: 28, font: 22 },
};

interface Props {
  icon?: string | null;
  size?: InitiativeIconSize;
}

export function InitiativeIcon({ icon, size = "sm" }: Props) {
  const { box, font } = SIZE[size];
  return (
    <View
      style={{
        width: box,
        height: box,
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <Text style={{ fontSize: font, lineHeight: Math.round(font * 1.2) }}>
        {icon || "🎯"}
      </Text>
    </View>
  );
}
