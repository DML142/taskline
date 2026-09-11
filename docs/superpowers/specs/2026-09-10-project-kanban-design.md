# Project Kanban Board Design

## Goal

Replace a project's flat issue list with a three-column Kanban board. Users can
quickly create an issue, assign it, filter the board, and move an issue between
To do, In progress, and Done without adding a drag-and-drop dependency.

## Scope and constraints

- The board is rendered by the existing project route:
  `apps/web/src/app/[workspaceSlug]/[projectSlug]/page.tsx`.
- Its columns map directly to the existing issue statuses `TODO`,
  `IN_PROGRESS`, and `DONE`; there is no custom status model or persisted card
  ordering in this phase.
- The client uses the native HTML Drag and Drop API. No drag-and-drop package
  is added.
- The existing issue-create and issue-update endpoints remain the only write
  APIs. A status move is an `IssueApi.update` request with the card's existing
  title, description, priority, and assignee plus its new status.
- The list API accepts independent optional `status`, `assigneeId`, and
  `priority` query parameters. Filtering is server-side so its result is
  consistent for all clients.
- A `VIEWER` cannot create or move issues. An `ADMIN` or `OWNER` can create,
  assign, and move every issue. A `MEMBER` can create an issue only for
  themselves or unassigned, and can move only an issue that is assigned to
  themselves; this is enforced by the existing service authorization rather
  than the browser.

## API and data flow

`ListFilter` gains a `Priority` field. The HTTP handler validates a non-empty
`priority` query parameter with the existing priority enum. The repository
selects the appropriate query for each supported combination of status,
assignee, and priority. A new SQL query covers all three filters, and the
generated sqlc Go code is regenerated. Invalid values return the established
400 `invalid_request` response.

The project page requests members, the project, and the filtered issue list as
it does today. It exposes three selects for assignee, priority, and status. A
status filter is intentionally a board-wide visibility filter: when selected,
the nonmatching columns are empty rather than relocating cards.

The quick-create control defaults to the To do column. It accepts title,
optional description, priority, and assignee. Admins and owners can choose any
workspace member; members see only themselves and the unassigned choice. A
successful submission prepends the returned issue to its status column.

Each card displays its title, priority, and assignee label and links to the
existing issue details page. Dragging is enabled only when the UI role makes a
move potentially legal: admin/owner for all cards and member for cards assigned
to the signed-in user. On drop into another column, the card moves
optimistically, receives the new status via `PATCH`, then replaces the local
card with the server response. A rejected request restores the prior board
state and displays the API error. Dropping into the card's current column does
nothing.

## Error handling and accessibility

The board retains the existing loading and error surfaces. Drag targets receive
an explicit visual state while a card is over them. Cards remain ordinary links
when not being dragged, so issue details stay reachable by keyboard. Native
drag-and-drop is used for this phase; keyboard reordering and persistent card
ordering are out of scope.

## Testing

- Go service and handler tests cover priority filtering, invalid priority, and
  member/admin authorization for a status-changing update.
- Web API tests cover construction of the `priority` query parameter.
- Project-page tests cover board grouping, all three filters, quick creation,
  permitted drag/drop, and rollback on a failed update.
- Run Go tests, vet, and build; web lint, typecheck, format check, and
  production build. Verify a successful browser drag-and-drop flow in Chromium.

## Deferred production hardening

Email verification, password reset, rate limiting, and deployment
configuration remain a separate later phase.
