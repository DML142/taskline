# Project Issues Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add authorized project issue tracking with API, UI, filters, and verification.

**Architecture:** A new `issue` API package follows the existing project package's service/repository/handler boundaries. SQL migrations and sqlc queries persist project-scoped issues. The Next.js project and issue routes consume a typed `IssueApi`.

**Tech Stack:** Go 1.27, pgx/sqlc/PostgreSQL, Next.js 16, React 19, TypeScript, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-07-project-issues-design.md`

## Global Constraints

- Keep the existing authenticated bearer-token API convention.
- Preserve workspace-role authorization in the service layer.
- Use `TODO`, `IN_PROGRESS`, and `DONE` statuses; use `LOW`, `MEDIUM`, and `HIGH` priorities.
- Add tests before production behavior and verify every test starts red.

---

### Task 1: Persist and expose issues

**Files:**

- Create: `apps/api/migrations/000004_issues.up.sql`, `apps/api/migrations/000004_issues.down.sql`, `apps/api/queries/issues.sql`, `apps/api/internal/issue/{service.go,repository.go,http.go}`
- Modify: `apps/api/cmd/api/main.go`, `apps/api/internal/platform/http/router.go`
- Test: `apps/api/internal/issue/{service_test.go,http_test.go}`

- [ ] Write service tests for validation, roles, and assignee membership; run them red.
- [ ] Add the migration, sqlc query definitions, and regenerate sqlc output.
- [ ] Add minimal issue service, repository, authenticated handler, and router wiring; run service and handler tests green.

### Task 2: Build project issue UI

**Files:**

- Create: `apps/web/src/lib/issue-api.ts`, `apps/web/src/lib/issue-api.test.ts`, `apps/web/src/app/[workspaceSlug]/[projectSlug]/[issueId]/page.tsx`
- Modify: `apps/web/src/app/[workspaceSlug]/[projectSlug]/page.tsx`
- Test: `apps/web/src/app/[workspaceSlug]/[projectSlug]/page.test.tsx`

- [ ] Write API and page tests for list/create/filter behaviour; run them red.
- [ ] Implement typed issue requests and project-page issue list, create form, and filters.
- [ ] Implement the issue detail page and role-gated edit form; run frontend tests green.

### Task 3: Verify the integrated feature

**Files:**

- Modify: `README.md`

- [ ] Document issue routes and supported workflow.
- [ ] Run Go formatting, tests, vet, build, sqlc generation/diff; run frontend tests, lint, typecheck, format check, and build.
- [ ] Inspect the final diff against the spec and correct every discovered discrepancy.
