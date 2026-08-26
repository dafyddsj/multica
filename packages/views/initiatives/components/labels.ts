"use client";

import type { InitiativeStatus, InitiativePriority } from "@multica/core/types";
import { useT } from "../../i18n";

export function useInitiativeStatusLabels(): Record<InitiativeStatus, string> {
  const { t } = useT("initiatives");
  return {
    planned: t(($) => $.status.planned),
    in_progress: t(($) => $.status.in_progress),
    paused: t(($) => $.status.paused),
    completed: t(($) => $.status.completed),
    cancelled: t(($) => $.status.cancelled),
  };
}

export function useInitiativePriorityLabels(): Record<InitiativePriority, string> {
  const { t } = useT("initiatives");
  return {
    urgent: t(($) => $.priority.urgent),
    high: t(($) => $.priority.high),
    medium: t(($) => $.priority.medium),
    low: t(($) => $.priority.low),
    none: t(($) => $.priority.none),
  };
}

export function useFormatRelativeDate(): (date: string) => string {
  const { t } = useT("initiatives");
  return (date: string) => {
    const diff = Date.now() - new Date(date).getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    if (days < 1) return t(($) => $.relative_date.today);
    if (days === 1) return t(($) => $.relative_date.one_day_ago);
    if (days < 30) return t(($) => $.relative_date.days_ago, { count: days });
    const months = Math.floor(days / 30);
    return t(($) => $.relative_date.months_ago, { count: months });
  };
}
