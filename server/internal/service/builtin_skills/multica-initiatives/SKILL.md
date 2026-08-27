---
name: multica-initiatives
description: "Use when creating, inspecting, updating, attaching projects to, or deleting a Multica initiative. An initiative is a durable parent of projects (product, service, program)."
user-invocable: false
allowed-tools: Bash(multica *)
---

# Multica Initiatives

## Quick start

```bash
multica initiative list --output json
multica initiative get <initiative-id> --output json
multica project list --output json
```

If the command shape is unclear, check help instead of guessing:

```bash
multica initiative --help
multica project create --help
multica project update --help
```

## Core model

An initiative is a durable parent above projects. Hierarchy is Workspace → Initiative → Project → Issue.

A project belongs to at most one initiative (`project.initiative_id`, nullable). Issues attach only through projects. There is no `issue.initiative_id`.

Status values default to `planned`, `in_progress`, `paused`, `completed`, `cancelled`. A workspace can add custom keys under Settings → Statuses; `--status` accepts those keys. `Active` is still rejected.

Priority values: `urgent`, `high`, `medium`, `low`, `none`.

Delete detaches child projects (nulls `project.initiative_id`). It never cascades. Any member can create or edit; only `owner` or `admin` can delete.

## CLI

```bash
multica initiative list --output json
multica initiative list --status in_progress --output json
multica initiative get <initiative-id> --output json
multica initiative create --title "<title>" --output json
multica initiative create --title "<title>" --start-date 2026-03-01 --due-date 2026-12-31 --output json
multica initiative update <initiative-id> --title "<title>" --output json
multica initiative update <initiative-id> --status in_progress --output json
multica initiative status <initiative-id> in_progress --output json
multica initiative delete <initiative-id>
multica project create --title "<title>" --initiative <initiative-id> --output json
multica project update <project-id> --initiative <initiative-id> --output json
multica project update <project-id> --initiative "" --output json
```

`--start-date` / `--due-date` are optional calendar days (`YYYY-MM-DD`). On `initiative update`, pass an empty string (`--start-date ""`) to clear a date; an unset flag leaves it untouched.

`--initiative ""` on `project update` detaches the project from its parent.

## Referring to an initiative in a comment

Use the mention-link form with the initiative UUID from `multica initiative list --output json`:

    [Platform](mention://initiative/<initiative-id>)

Web and desktop render a chip. Mobile renders an ordinary link. Like a project mention, this is a pure link with no notification side effect.

## Side effects

Initiative create/update/delete/status mutate durable workspace state. Delete detaches projects; it does not delete them. Ask before deleting.

More source-backed details: `references/initiatives-source-map.md`.
