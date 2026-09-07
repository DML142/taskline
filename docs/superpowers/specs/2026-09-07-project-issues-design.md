# Project Issues Design

## Goal

Add project-scoped issues that workspace members can create, read, filter, and update through the API and web UI.

## Data model

An issue belongs to one project. It has a UUID, title (1–200 characters), description (up to 10,000 characters), status (`TODO`, `IN_PROGRESS`, `DONE`), priority (`LOW`, `MEDIUM`, `HIGH`), a required creator, an optional assignee, and timestamps. An assignee must be a member of the issue's workspace. Project deletion cascades to its issues.

## Authorization

All workspace members can list and read issues. Owners and administrators can create and update any issue. Members can create issues and update only issues assigned to themselves; they cannot change an assignee to another user. Viewers cannot create or update issues. These checks live in the service layer.

## API

Routes are authenticated and use a workspace UUID plus a project slug:

- `GET /api/v1/issues/{workspaceID}/{projectSlug}` lists issues and accepts optional `status` and `assigneeId` query filters.
- `POST /api/v1/issues/{workspaceID}/{projectSlug}` creates an issue.
- `GET /api/v1/issues/{workspaceID}/{projectSlug}/{issueID}` returns one issue.
- `PATCH /api/v1/issues/{workspaceID}/{projectSlug}/{issueID}` updates an issue.

Responses use the existing `{ "issue": ... }` or `{ "issues": [...] }` conventions. Invalid payloads and filters return `400`, missing resources return `404`, prohibited actions return `403`, and unknown assignees return `400`.

## Web UI

The project page shows a create form for permitted users, issue rows with status and priority, and status/assignee filters. Each row navigates to a dedicated issue route. The detail page shows the issue and an edit form only when the current workspace role permits the requested changes.

## Verification

Service tests prove role handling, validation, and assignee membership. HTTP tests prove route responses. Web API tests prove request construction, and page tests prove issue rendering and filtering. Full Go tests, frontend tests, lint, typecheck, and production build run before completion.
