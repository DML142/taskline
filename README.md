# Taskline

Issue tracking and project management application built with Go, PostgreSQL and Next.js.

The repository includes email/password authentication, session rotation, workspaces, role-based workspace membership, projects, and project issues.

## Stack

- Go 1.27.1, net/http, chi and slog
- Next.js 16, React 19, strict TypeScript, Tailwind CSS and shadcn/ui with Base UI
- PostgreSQL 18 through Docker Compose
- Go modules for the API; pnpm workspace for `apps/web`

## Local development

Requirements: Go 1.27.1 (or automatic Go toolchain downloads enabled), Node.js 22.12+ and pnpm 11.20.0, Docker with Compose.

From the repository root:

```sh
cp .env.example .env
docker compose up -d --wait postgres
pnpm install --frozen-lockfile
```

Start the API in a separate terminal:

```sh
cd apps/api
export DATABASE_URL='postgres://taskline:taskline-local@127.0.0.1:5432/taskline?sslmode=disable'
export JWT_SECRET='replace-this-with-a-random-secret-of-at-least-32-bytes'
go run ./cmd/migrate up
go run ./cmd/api
```

Migrations run only through the explicit `go run ./cmd/migrate up` command; API startup never applies them automatically. Run migration commands from `apps/api`. The available commands are `up`, `down` and `version`.

Start the frontend from the repository root:

```sh
pnpm dev
```

Open http://localhost:3000 and create an account. The frontend communicates directly with the API at `http://localhost:8080/api/v1` by default.

```sh
curl http://127.0.0.1:8080/api/v1/health
curl http://127.0.0.1:8080/api/v1/ready
```

Both endpoints return `{"status":"ok"}` when successful. `/api/v1/health` reports process liveness and does not check PostgreSQL. `/api/v1/ready` reports whether the API can reach PostgreSQL; it returns HTTP 503 with a `database_unavailable` JSON error when the database ping fails.

## Configuration

Root `.env` configures Compose only: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` and `POSTGRES_PORT`. The example password is for local development only. PostgreSQL binds to loopback and persists data in the `taskline_postgres_data` volume.

The API reads `DATABASE_URL`, `JWT_SECRET`, `JWT_ISSUER`, `JWT_AUDIENCE`, `WEB_ORIGIN`, `COOKIE_SECURE` and `HTTP_ADDR` from the process environment; it does not load dotenv files. `DATABASE_URL` and a `JWT_SECRET` of at least 32 bytes are required. `HTTP_ADDR` defaults to `127.0.0.1:8080`, `JWT_ISSUER` to `taskline-api`, `JWT_AUDIENCE` to `taskline-web`, `WEB_ORIGIN` to `http://localhost:3000`, and `COOKIE_SECURE` to `false`. See `apps/api/.env.example`.

Workspace invitations require `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, and `SMTP_FROM`. The configured server must support STARTTLS; use port 587 unless your provider documents another STARTTLS port. `WEB_ORIGIN` must be the public application origin because invitation emails contain a one-time acceptance URL.

```sh
cd apps/api
export DATABASE_URL='postgres://taskline:taskline-local@127.0.0.1:5432/taskline?sslmode=disable'
export JWT_SECRET='replace-this-with-a-random-secret-of-at-least-32-bytes'
HTTP_ADDR=127.0.0.1:9090 go run ./cmd/api
```

Set `NEXT_PUBLIC_API_URL` only when the API is not at `http://localhost:8080/api/v1`.

## Authentication

`POST /api/v1/auth/register`, `/login`, `/refresh` and `/logout` manage short-lived access JWTs and rotated HttpOnly refresh sessions. `GET /api/v1/auth/me` requires a bearer access token. Password reset, email verification, OAuth and MFA are not part of this phase.

After changing SQL queries or migrations, regenerate the checked-in database package from `apps/api` with the pinned sqlc tool:

```sh
go tool sqlc generate
```

## Workspaces

Authenticated users can create and list their workspaces. Each creator is an `OWNER`; workspace members have one of `OWNER`, `ADMIN`, `MEMBER`, or `VIEWER` roles. Every member can read the workspace and its member list. In this phase, only owners can rename or delete a workspace, or add and remove members. Members must already have an account; invitations and ownership transfer are not implemented.

Workspace routes are protected by bearer access tokens under `/api/v1/workspaces`. Apply migrations explicitly before using them:

```sh
cd apps/api
go run ./cmd/migrate up
```

Owners and administrators invite people by email. Each opaque invitation URL expires after 24 hours, can be accepted only once, and requires the recipient to sign in or register using the exact invited email address. Owners can remove any other member; administrators can remove members/viewers and only administrators whom they added.

## Projects

Each workspace and project has a human-readable, URL-safe slug. The frontend uses GitHub-style routes such as `/acme/website-redesign`; UUIDs remain the internal API identifiers. Existing workspaces receive a stable `workspace-<uuid-prefix>` slug when migration `000003_projects` is applied.

Workspace `OWNER`s and `ADMIN`s can create, edit and archive projects. `MEMBER`s and `VIEWER`s can read projects. Project slug values are unique within a workspace and use lowercase letters, digits and single hyphens. The project API is protected by bearer authentication:

```text
GET  /api/v1/projects/{workspaceUUID}
POST /api/v1/projects/{workspaceUUID}
GET  /api/v1/projects/{workspaceUUID}/{projectSlug}
PATCH /api/v1/projects/{workspaceUUID}/{projectSlug}
```

## Issues

Projects contain issues with a title, description, status (`TODO`, `IN_PROGRESS`, or `DONE`), priority (`LOW`, `MEDIUM`, or `HIGH`), creator, and optional assignee. Owners and admins can manage every issue. Members can create issues and update issues assigned to themselves; viewers can read only. Assignees must be workspace members; removing a member automatically clears their issue assignments.

Issue routes use a workspace UUID and project slug. The list endpoint supports optional `status` and `assigneeId` filters:

```text
GET   /api/v1/issues/{workspaceUUID}/{projectSlug}?status=TODO&assigneeId={userUUID}
POST  /api/v1/issues/{workspaceUUID}/{projectSlug}
GET   /api/v1/issues/{workspaceUUID}/{projectSlug}/{issueUUID}
PATCH /api/v1/issues/{workspaceUUID}/{projectSlug}/{issueUUID}
```

## Checks

From `apps/api`:

```sh
gofmt -l .
go vet ./...
go test ./...
go build ./...
golangci-lint run ./...
go tool sqlc generate
# After the initial Git baseline, confirms generated sqlc code is current.
git diff --exit-code -- internal/platform/database/sqlc
# With PostgreSQL from Compose running and DATABASE_URL exported.
go run ./cmd/migrate version
```

Install [golangci-lint v2.13.2](https://github.com/golangci/golangci-lint/releases/tag/v2.13.2), built with Go 1.27 support.

From the repository root:

```sh
pnpm lint
pnpm typecheck
pnpm format:check
pnpm --filter @taskline/web exec next build --webpack
docker compose --env-file .env.example config --quiet
# Checks the local Compose database using its documented development credentials.
docker compose exec -T postgres psql -U taskline -d taskline -c 'SELECT 1;'
```

Use `pnpm format` for frontend and configuration formatting, and `gofmt -w .` from `apps/api` for Go.

## License

[MIT](LICENSE)
