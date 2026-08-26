import type { Initiative } from "@multica/core/types";
import { cn } from "@multica/ui/lib/utils";

export type InitiativeIconSize = "sm" | "md" | "lg";

export interface InitiativeIconProps {
  initiative?: Pick<Initiative, "icon"> | null;
  size?: InitiativeIconSize;
  className?: string;
}

const SIZE_CLASS: Record<InitiativeIconSize, string> = {
  sm: "size-3.5 text-caption leading-none",
  md: "size-4 text-body leading-none",
  lg: "size-6 text-display-sm leading-none",
};

export function InitiativeIcon({ initiative, size = "sm", className }: InitiativeIconProps) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "inline-flex shrink-0 items-center justify-center",
        SIZE_CLASS[size],
        className,
      )}
    >
      {initiative?.icon || "🎯"}
    </span>
  );
}
