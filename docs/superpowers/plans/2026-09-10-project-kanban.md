# Project Kanban Board Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace a project's issue list with a three-column Kanban board that supports native drag-and-drop status updates, quick creation and assignment, and filters by assignee, priority, and status.

**Architecture:** Retain the existing issue routes and service authorization. Extend the list filter through HTTP, service, repository, SQL query, and generated sqlc code; the web route renders the returned issues in fixed status columns and moves cards with the existing complete-update endpoint.

**Tech Stack:** Go 1.24, PostgreSQL, sqlc, Next.js 16, React 19, TypeScript, Tailwind CSS, Vitest, Testing Library, native HTML Drag and Drop API.

**Spec:** `docs/superpowers/specs/2026-09-10-project-kanban-design.md`

## Global Constraints

- Do not add a drag-and-drop dependency; use the browser's native Drag and Drop API.
- Preserve the existing `VIEWER`, `MEMBER`, `ADMIN`, and `OWNER` authorization behavior.
- A member can move only an issue currently assigned to that member; admins and owners can move every issue.
- Filters are server-side and may be combined: `status`, `assigneeId`, and `priority`.
- Do not introduce card ordering, custom columns, keyboard drag-and-drop, or production-hardening work in this change.

---

### Task 1: Add priority filtering to the issue API

**Files:**
- Modify: `apps/api/queries/issues.sql`
- Modify: `apps/api/internal/platform/database/sqlc/issues.sql.go`
- Modify: `apps/api/internal/issue/service.go`
- Modify: `apps/api/internal/issue/repository.go`
- Modify: `apps/api/internal/issue/http.go`
- Test: `apps/api/internal/issue/http_test.go`
- Test: `apps/api/internal/issue/service_test.go`

**Interfaces:**
- Consumes: `Priority` (`LOW | MEDIUM | HIGH`), `ListFilter`, and `IssueApi.list` query conventions.
- Produces: `ListFilter{Status Status, AssigneeID *uuid.UUID, Priority Priority}` and `GET /issues/{workspaceID}/{projectSlug}?status=&assigneeId=&priority=`.

- [ ] **Step 1: Write failing HTTP parsing tests**

```go
func TestListFilterParsesStatusAssigneeAndPriority(t *testing.T) {
    assigneeID := uuid.New()
    request := httptest.NewRequest("GET", "/?status=IN_PROGRESS&assigneeId="+assigneeID.String()+"&priority=HIGH", nil)

    filter, err := listFilter(request)

    require.NoError(t, err)
    require.Equal(t, StatusInProgress, filter.Status)
    require.Equal(t, &assigneeID, filter.AssigneeID)
    require.Equal(t, PriorityHigh, filter.Priority)
}

func TestListFilterRejectsInvalidPriority(t *testing.T) {
    _, err := listFilter(httptest.NewRequest("GET", "/?priority=URGENT", nil))
    require.ErrorIs(t, err, ErrInvalidRequest)
}
```

- [ ] **Step 2: Run the handler tests to verify they fail**

Run: `go test ./apps/api/internal/issue -run 'TestListFilter(ParsesStatusAssigneeAndPriority|RejectsInvalidPriority)' -count=1`

Expected: FAIL because `ListFilter` has no `Priority` field and `listFilter` does not validate priority.

- [ ] **Step 3: Write a failing service test for a valid priority filter**

```go
func TestListAcceptsPriorityFilter(t *testing.T) {
    repository := &fakeRepository{role: workspace.RoleViewer}
    service := NewService(repository)

    _, err := service.List(context.Background(), uuid.New(), uuid.New(), "website-redesign", ListFilter{Priority: PriorityHigh})

    require.NoError(t, err)
    require.Equal(t, PriorityHigh, repository.listFilter.Priority)
}
```

Add `listFilter ListFilter` to `fakeRepository` and store it in its `List` method.

- [ ] **Step 4: Run the service test to verify it fails**

Run: `go test ./apps/api/internal/issue -run TestListAcceptsPriorityFilter -count=1`

Expected: FAIL because `ListFilter` has no `Priority` field.

- [ ] **Step 5: Add the database query and regenerate sqlc**

Append this query to `apps/api/queries/issues.sql`:

```sql
-- name: ListIssuesForProjectByFilters :many
SELECT id, project_id, title, description, status, priority, creator_id, assignee_id, created_at, updated_at
FROM issues
WHERE project_id = $1
  AND ($2::text = '' OR status = $2)
  AND ($3::uuid IS NULL OR assignee_id = $3)
  AND ($4::text = '' OR priority = $4)
ORDER BY created_at DESC, id DESC;
```

Run: `(cd apps/api && go run github.com/sqlc-dev/sqlc/cmd/sqlc generate)`

Keep the generated `ListIssuesForProjectByFilters` method in `apps/api/internal/platform/database/sqlc/issues.sql.go` checked in.

- [ ] **Step 6: Implement parsing, validation, and repository delegation**

Update `ListFilter` and `Service.List`:

```go
type ListFilter struct {
    Status Status
    AssigneeID *uuid.UUID
    Priority Priority
}

if filter.Priority != "" && !validPriority(filter.Priority) {
    return nil, ErrInvalidRequest
}
```

Parse `priority` in `listFilter`, returning `ErrInvalidRequest` unless it is empty or a valid priority. Replace the repository's branch matrix with:

```go
rows, err := database.New(r.pool).ListIssuesForProjectByFilters(ctx,
    database.ListIssuesForProjectByFiltersParams{
        ProjectID: projectID,
        Status: string(filter.Status),
        AssigneeID: nullableUUID(filter.AssigneeID),
        Priority: string(filter.Priority),
    },
)
```

- [ ] **Step 7: Run focused Go tests to verify they pass**

Run: `go test ./apps/api/internal/issue -count=1`

Expected: PASS, including parsing, validation, list delegation, and existing member status-update authorization tests.

- [ ] **Step 8: Commit the API filter work**

```bash
git add apps/api/queries/issues.sql apps/api/internal/platform/database/sqlc/issues.sql.go apps/api/internal/issue/service.go apps/api/internal/issue/repository.go apps/api/internal/issue/http.go apps/api/internal/issue/http_test.go apps/api/internal/issue/service_test.go
git commit -m "feat: filter project issues by priority"
```

### Task 2: Expose the priority filter in the web API client

**Files:**
- Modify: `apps/web/src/lib/issue-api.ts`
- Test: `apps/web/src/lib/issue-api.test.ts`

**Interfaces:**
- Consumes: `IssuePriority` and `IssueFilter` from `issue-api.ts`.
- Produces: `IssueApi.list(workspaceId, projectSlug, { status?, assigneeId?, priority? })` encoding the three optional query parameters.

- [ ] **Step 1: Write the failing client test**

Change the list test's input and expected URL to:

```ts
await api.list("workspace-1", "website", {
  status: "IN_PROGRESS",
  assigneeId: "user-1",
  priority: "HIGH",
});

expect(request).toHaveBeenCalledWith(
  "/api/v1/issues/workspace-1/website?status=IN_PROGRESS&assigneeId=user-1&priority=HIGH",
);
```

- [ ] **Step 2: Run the client test to verify it fails**

Run: `pnpm --dir apps/web test src/lib/issue-api.test.ts`

Expected: FAIL because `IssueFilter` does not accept `priority` and the URL omits it.

- [ ] **Step 3: Implement the smallest API-client change**

Add `priority?: IssuePriority` to `IssueFilter` and append it in `IssueApi.list`:

```ts
if (filter.priority) params.set("priority", filter.priority);
```

- [ ] **Step 4: Run the client test to verify it passes**

Run: `pnpm --dir apps/web test src/lib/issue-api.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit the client change**

```bash
git add apps/web/src/lib/issue-api.ts apps/web/src/lib/issue-api.test.ts
git commit -m "feat: expose issue priority filter"
```

### Task 3: Replace the issue list with a native Kanban board

**Files:**
- Modify: `apps/web/src/app/[workspaceSlug]/[projectSlug]/page.tsx`
- Modify: `apps/web/src/app/[workspaceSlug]/[projectSlug]/page.test.tsx`

**Interfaces:**
- Consumes: `IssueApi.create`, `IssueApi.update`, the extended `IssueApi.list`, `WorkspaceMember`, and current authenticated user id.
- Produces: a three-column board with labelled drop targets; cards invoke `IssueApi.update` with a complete `IssueInput` and a replacement `status`.

- [ ] **Step 1: Write failing board-rendering tests**

Return one issue per status from the page test's issue stub and assert fixed column headings and card placement:

```tsx
expect(await screen.findByRole("heading", { name: "To do" })).toBeTruthy();
expect(screen.getByRole("heading", { name: "In progress" })).toBeTruthy();
expect(screen.getByRole("heading", { name: "Done" })).toBeTruthy();
expect(screen.getByText("Ship the redesign")).toBeTruthy();
expect(screen.getByLabelText("Priority")).toBeTruthy();
```

Use `getByLabelText("To do")`, `getByLabelText("In progress")`, and `getByLabelText("Done")` for column drop zones so placement can be tested by DOM containment.

- [ ] **Step 2: Run the project-page test to verify it fails**

Run: `pnpm --dir apps/web test 'src/app/[workspaceSlug]/[projectSlug]/page.test.tsx'`

Expected: FAIL because the current route renders a flat issue list and no priority filter.

- [ ] **Step 3: Write failing interaction tests for filters, creation, and status movement**

Use `userEvent.selectOptions` to choose priority and assert that the issue request includes `priority=HIGH`. Submit the quick-create form and assert the returned card appears in To do. Dispatch `dragStart`, `dragOver`, and `drop` from a draggable card to the In progress zone; assert its request is a `PATCH` with unchanged title/description/priority/assignee and `status: "IN_PROGRESS"`. Make that PATCH return 403 in a separate test and assert the card remains in its original column and an alert appears.

- [ ] **Step 4: Run interaction tests to verify they fail**

Run: `pnpm --dir apps/web test 'src/app/[workspaceSlug]/[projectSlug]/page.test.tsx'`

Expected: FAIL because no board handlers, priority filter, or optimistic rollback behavior exists.

- [ ] **Step 5: Implement board state and server-side filter controls**

Add `priorityFilter` state of type `"" | IssuePriority`; pass it to `issueApi.list` together with `statusFilter` and `assigneeFilter`. Render a `Priority` select with All priorities, Low, Medium, and High. Define the fixed column metadata:

```ts
const boardColumns: ReadonlyArray<{ status: IssueStatus; title: string }> = [
  { status: "TODO", title: "To do" },
  { status: "IN_PROGRESS", title: "In progress" },
  { status: "DONE", title: "Done" },
];
```

Render one labelled `section` for every column, filtering `issues` by its status. Preserve the existing detail-page `Link` per issue, and include priority and resolved member name in each card.

- [ ] **Step 6: Implement quick creation and native drag-and-drop**

Keep the existing form but remove its status select and always submit `status: "TODO"`; retain priority and role-aware assignment. Add `draggedIssueId` and `activeDropStatus` state. A card is draggable when:

```ts
const canMoveIssue = canManage || issue.assigneeId === user?.id;
```

On `dragStart`, set the issue id in both React state and `event.dataTransfer` (`text/plain`). On a column `dragOver`, call `event.preventDefault()` only if a draggable card is active. On `drop`, ignore empty, same-status, or unauthorized drops; otherwise replace the issue in local state with `{ ...issue, status: targetStatus }`, call `issueApi.update` with all existing issue fields plus `status: targetStatus`, replace the optimistic card with the response on success, and restore the captured array plus a readable error on failure. Clear drag state on `dragEnd` and after every drop.

- [ ] **Step 7: Run the project-page tests to verify they pass**

Run: `pnpm --dir apps/web test 'src/app/[workspaceSlug]/[projectSlug]/page.test.tsx'`

Expected: PASS for headings, grouping, filters, creation, allowed drag/drop, and rejected-update rollback.

- [ ] **Step 8: Commit the board work**

```bash
git add 'apps/web/src/app/[workspaceSlug]/[projectSlug]/page.tsx' 'apps/web/src/app/[workspaceSlug]/[projectSlug]/page.test.tsx'
git commit -m "feat: add project kanban board"
```

### Task 4: Run the complete verification suite and inspect the browser workflow

**Files:**
- Modify only if verification reveals a defect in a file listed above.

**Interfaces:**
- Consumes: completed API and web changes.
- Produces: evidence that formatting, static analysis, tests, builds, and the actual browser workflow work together.

- [ ] **Step 1: Run backend verification**

Run:

```bash
(cd apps/api && go test ./...)
(cd apps/api && go vet ./...)
(cd apps/api && go build ./...)
```

Expected: all commands exit 0.

- [ ] **Step 2: Run web verification**

Run:

```bash
pnpm --dir apps/web test
pnpm lint
pnpm typecheck
pnpm format:check
pnpm build
```

Expected: all commands exit 0.

- [ ] **Step 3: Exercise the real browser flow**

With the local API, PostgreSQL, and web dev server running, sign in as an owner; open a project; create a To do issue with a priority and assignee; drag it to In progress; select each filter; verify the card's updated status after a reload. Capture the result in the final handoff.

- [ ] **Step 4: Commit any verification-only correction**

```bash
git add apps/api apps/web
git commit -m "fix: verify project kanban workflow"
```

Only create this commit if Step 1–3 required a correction; otherwise do not create an empty commit.

