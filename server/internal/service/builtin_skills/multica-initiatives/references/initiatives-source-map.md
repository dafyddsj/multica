# Initiatives source map

- `server/cmd/multica/cmd_initiative.go` registers initiative `list`, `get`, `create`, `update`, `delete`, and `status`.
- Initiative and project status keys are a per-workspace catalog (`entity_status`, `/api/entity-statuses`). Built-ins are `planned`, `in_progress`, `paused`, `completed`, `cancelled`; custom keys inherit a category. `validateProjectStatus` / `resolveWritableEntityStatus` accept format-valid custom keys.
- `server/cmd/multica/cmd_project.go` accepts `--initiative` on `project create` and `project update` (empty string on update detaches).
- `server/cmd/multica/cmd_id_resolver.go` resolves initiative ids by UUID prefix or title via `GET /api/initiatives`.
- `server/cmd/server/router.go` exposes `/api/initiatives` plus `/api/initiatives/search` and `/api/initiatives/{id}`.
- `server/pkg/db/queries/initiative.sql` is the CRUD and stats query surface for `initiative` rows.
- `project.initiative_id` is a nullable column with no database foreign key. `resolveInitiativeInWorkspace` in `server/internal/handler/initiative.go` validates the parent on project create/update.
- Delete (`DELETE /api/initiatives/{id}`) nulls `project.initiative_id` for children and removes initiative pins. It does not delete projects or issues.
- Pins of type `initiative` are withheld unless the client sends `include=initiative` (`server/internal/handler/pin.go`). Old clients treat unknown pin types as projects and auto-unpin on 404.
- Referring to an initiative in a comment: `[Label](mention://initiative/<uuid>)` is a render-only link, same no-side-effect contract as a project mention.
