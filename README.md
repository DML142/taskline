# Taskline

Issue tracking and project management application built with Go, PostgreSQL and Next.js.

The repository currently contains an API health endpoint and a minimal application shell. Authentication, workspaces, projects and issues are planned.

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
go run ./cmd/migrate up
go run ./cmd/api
```

Migrations run only through the explicit `go run ./cmd/migrate up` command; API startup never applies them automatically. Run migration commands from `apps/api`. The available commands are `up`, `down` and `version`.

Start the frontend from the repository root:

```sh
pnpm dev
```

Open http://localhost:3000. The frontend shell currently runs independently of the API.

```sh
curl http://127.0.0.1:8080/api/v1/health
curl http://127.0.0.1:8080/api/v1/ready
```

Both endpoints return `{"status":"ok"}` when successful. `/api/v1/health` reports process liveness and does not check PostgreSQL. `/api/v1/ready` reports whether the API can reach PostgreSQL; it returns HTTP 503 with a `database_unavailable` JSON error when the database ping fails.

## Configuration

Root `.env` configures Compose only: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` and `POSTGRES_PORT`. The example password is for local development only. PostgreSQL binds to loopback and persists data in the `taskline_postgres_data` volume.

The API reads `DATABASE_URL` and `HTTP_ADDR` from the process environment; it does not load dotenv files. `DATABASE_URL` is required, while `HTTP_ADDR` defaults to `127.0.0.1:8080`. See `apps/api/.env.example`. Export the database URL before running either the API or migration command:

```sh
cd apps/api
export DATABASE_URL='postgres://taskline:taskline-local@127.0.0.1:5432/taskline?sslmode=disable'
HTTP_ADDR=127.0.0.1:9090 go run ./cmd/api
```

No frontend environment variables are required yet.

After changing SQL queries or migrations, regenerate the checked-in database package from `apps/api` with the pinned sqlc tool:

```sh
go tool sqlc generate
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
