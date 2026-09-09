# Workspace Invitations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver email-bound, single-use workspace invitations over SMTP and enforce the approved owner/admin hierarchy.

**Architecture:** `workspace` remains the source of truth for membership and removal authorization. A new `invite` package creates opaque tokens, persists their SHA-256 hashes, sends email via an SMTP adapter, and accepts invitations transactionally. Membership provenance is recorded in `workspace_members.added_by_user_id` and returned to authorized web users.

**Tech Stack:** Go 1.27, PostgreSQL 18, pgx/sqlc, `net/smtp` with STARTTLS, chi/net/http, Next.js 16, React 19, TypeScript, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-09-workspace-invitations-design.md`

## Global Constraints

- URLs contain only a cryptographically random opaque secret; never include an email, role, workspace ID, or predictable code.
- Persist only `sha256(secret)`. An invite expires 24 hours after creation, is single-use, and is bound to the normalized account email.
- `OWNER` may manage any non-owner. `ADMIN` may manage members/viewers and only administrators whose `added_by_user_id` equals the acting admin.
- Keep role changes owner-only. They must not manufacture administrator provenance.
- Do not log or return SMTP credentials, recipient emails, or raw invitation secrets.
- Write each new test first, run it red, then implement the smallest code to make it green.

---

### Task 1: Persist invitations and provenance

**Files:**

- Create: `apps/api/migrations/000006_workspace_invites.up.sql`, `apps/api/migrations/000006_workspace_invites.down.sql`, `apps/api/queries/invites.sql`, `apps/api/internal/invite/repository.go`, `apps/api/internal/invite/repository_test.go`
- Modify: `apps/api/queries/workspaces.sql`, `apps/api/internal/workspace/{repository.go,repository_test.go}`, `apps/api/internal/platform/database/sqlc/{models.go,workspaces.sql.go}`

**Interfaces:**

- `workspace.Membership` exposes `AddedByUserID *uuid.UUID`; `workspace.Member` additionally exposes `AddedByName *string`.
- `invite.Repository.Accept(ctx context.Context, tokenHash [32]byte, userID uuid.UUID, normalizedEmail string) (workspace.Membership, error)` locks the invite and consumes it atomically.

- [ ] **Step 1: Write failing repository tests**

Create users for an owner, inviter, and invitee. Assert accepting an invite creates the role requested and writes the inviter ID as provenance. Add separate cases for a second acceptance, expired invite, revoked invite, and normalized-email mismatch; each must fail and not create a membership.

```go
membership, err := repository.Accept(ctx, hashToken(secret), invitee.ID, "invitee@example.com")
require.NoError(t, err)
require.Equal(t, inviter.ID, *membership.AddedByUserID)
_, err = repository.Accept(ctx, hashToken(secret), invitee.ID, "invitee@example.com")
require.ErrorIs(t, err, invite.ErrUnavailable)
```

- [ ] **Step 2: Run them red**

Run: `cd apps/api && DATABASE_URL="$DATABASE_URL" go test ./internal/invite ./internal/workspace -run 'Test.*(Invite|Provenance)' -count=1`

Expected: FAIL because invite persistence and provenance do not exist.

- [ ] **Step 3: Add database migration and query definitions**

Add nullable `workspace_members.added_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL` and an index on `(workspace_id, added_by_user_id)`. Create `workspace_invites` with ID, workspace ID, normalized email, target `workspace_role`, unique `token_hash BYTEA`, creator ID, expiry, optional accepted/revoked timestamps, and timestamps. The down migration drops these in reverse order. Define SQLC queries for create, lock-by-token-hash, list active, revoke, revoke active rows for an email, and atomic acceptance. Extend workspace member create/list queries with provenance and adder name.

```sql
CREATE UNIQUE INDEX workspace_invites_one_active_email_idx
ON workspace_invites (workspace_id, email)
WHERE accepted_at IS NULL AND revoked_at IS NULL;
```

- [ ] **Step 4: Generate SQLC and implement transaction behavior**

Run `cd apps/api && go tool sqlc generate`. Implement a `pgx.Tx` acceptance method: lock invite by hash; check expiry, consumption, revocation, and email; insert the membership with `added_by_user_id`; set `accepted_at`; commit. Before creating an invite, revoke previous rows for that email including expired unrevoked rows, then insert so the immutable partial unique index remains valid. Map unavailable states to sentinel errors and do not consume a mismatched-email invite.

- [ ] **Step 5: Verify and commit**

Run: `cd apps/api && DATABASE_URL="$DATABASE_URL" go test ./internal/invite ./internal/workspace -run 'Test.*(Invite|Provenance)' -count=1`

Expected: PASS.

Commit: `git add apps/api/migrations apps/api/queries apps/api/internal/{invite,workspace} apps/api/internal/platform/database/sqlc && git commit -m "feat: persist workspace invitations"`

### Task 2: Add authorization, token generation, SMTP delivery, and configuration

**Files:**

- Create: `apps/api/internal/invite/{service.go,mailer.go,service_test.go,mailer_test.go}`
- Modify: `apps/api/internal/workspace/{service.go,service_test.go}`, `apps/api/internal/platform/config/{config.go,config_test.go}`, `apps/api/cmd/api/main.go`

**Interfaces:**

- `type Mailer interface { Send(context.Context, Message) error }` and `NewSMTPMailer(config.SMTPConfig) *SMTPMailer`.
- `invite.Service` exposes `Create`, `Preview`, `Accept`, `ListActive`, and `Revoke`.
- `config.Config` has `SMTP config.SMTPConfig` with `Host`, `Port`, `Username`, `Password`, and `From`.

- [ ] **Step 1: Write failing policy, service, mailer, and config tests**

Add table-driven removal-policy tests: owner removes any non-owner; admin removes member/viewer; admin removes an admin it invited; admin cannot remove self, owner, peer admin, or an admin added by another person. Use a recording fake mailer to test normalized email, 24-hour expiry, replacement/revocation, creator-filtered list/revoke, and SMTP rollback. Validate all five SMTP environment variables and malformed port/sender cases.

```go
require.NoError(t, workspaces.RemoveMember(ctx, inviter.ID, workspace.ID, invitedAdmin.ID))
require.ErrorIs(t, workspaces.RemoveMember(ctx, peerAdmin.ID, workspace.ID, invitedAdmin.ID), workspace.ErrForbidden)
mailer := &recordingMailer{err: errors.New("SMTP unavailable")}
require.ErrorIs(t, createInviteWith(mailer), invite.ErrDeliveryFailed)
```

- [ ] **Step 2: Run tests red**

Run: `cd apps/api && go test ./internal/invite ./internal/workspace ./internal/platform/config -run 'Test.*(Invite|Admin|SMTP|Member|Load)' -count=1`

Expected: FAIL because services, mailer, config, and policy are absent.

- [ ] **Step 3: Implement configuration and SMTP adapter**

Load required `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, and `SMTP_FROM`; validate port range and sender email. Implement a STARTTLS SMTP client using the configured host as the TLS server name and `smtp.PlainAuth`. The text email uses `${WEB_ORIGIN}/invites/<secret>` and its expiration; wrap transport errors without secret values.

```go
type Message struct { To, WorkspaceName, AcceptanceURL string; ExpiresAt time.Time }
type Mailer interface { Send(context.Context, Message) error }
func NewSMTPMailer(cfg config.SMTPConfig) *SMTPMailer
```

- [ ] **Step 4: Implement invitation and membership services**

Create 32 random bytes with `crypto/rand`, encode with URL-safe base64 without padding, and hash with `sha256.Sum256`. Create/revoke/list invitations according to actor role; admin list/revoke queries include its creator ID. Make delivery and invite persistence atomic: transaction creates/replaces the record, mail is sent, then transaction commits; delivery failure rolls it back. Move deletion authorization into `workspace.canRemoveMember`; remove direct member addition from the HTTP path.

```go
func (s *Service) Create(ctx context.Context, actorID, workspaceID uuid.UUID, email string, role workspace.Role) (Invite, error)
func (s *Service) Preview(ctx context.Context, rawToken string) (Preview, error)
func (s *Service) Accept(ctx context.Context, rawToken string, identity auth.Identity) (workspace.Membership, error)
```

- [ ] **Step 5: Verify and commit**

Run: `cd apps/api && go test ./internal/invite ./internal/workspace ./internal/platform/config -run 'Test.*(Invite|Admin|SMTP|Member|Load)' -count=1`

Expected: PASS with no live SMTP connection.

Commit: `git add apps/api/internal/invite apps/api/internal/workspace apps/api/internal/platform/config apps/api/cmd/api/main.go && git commit -m "feat: deliver workspace invitations by email"`

### Task 3: Expose secure invitation HTTP routes

**Files:**

- Create: `apps/api/internal/invite/{http.go,http_test.go}`
- Modify: `apps/api/internal/workspace/{http.go,http_test.go}`, `apps/api/internal/platform/http/{router.go,router_test.go}`, `apps/api/cmd/api/main.go`

**Interfaces:**

- `POST|GET /api/v1/workspaces/{workspaceID}/invites` and `DELETE /api/v1/workspaces/{workspaceID}/invites/{inviteID}` require bearer authentication.
- `GET|POST /api/v1/workspace-invites/{token}` previews and accepts tokens; both require bearer authentication.
- `GET /members` returns `addedByUserId` and `addedByName`; `POST /members` is no longer mounted.

- [ ] **Step 1: Write failing handler and router tests**

Assert invite creation returns `201`; admin sees only its active invitations; owner sees all; unauthorized revocation returns `403`. Assert invalid, expired, revoked, and used tokens receive identical `404` JSON without recipient email. Assert mismatch returns `403` without recipient email, member responses return provenance, and `POST /members` is method-not-allowed.

```go
require.JSONEq(t, `{"error":{"code":"invite_not_found","message":"Invitation is unavailable"}}`, response.Body.String())
require.NotContains(t, response.Body.String(), "invited@example.com")
```

- [ ] **Step 2: Run tests red**

Run: `cd apps/api && go test ./internal/invite ./internal/workspace ./internal/platform/http -run 'Test.*(Invite|Member|Router)' -count=1`

Expected: FAIL because routes and safe error mappings are missing.

- [ ] **Step 3: Implement handlers and wire routes**

Use strict JSON `{ "email": string, "role": Role }` with existing size limits. Mount workspace invite endpoints from the workspace handler and opaque-token endpoints from the invite handler. Replace the router's positional handler variadic with named workspace/project/issue/invite fields or a struct so the new mount is explicit. Map unavailable token states to one generic error; never write raw token or invitation email.

- [ ] **Step 4: Verify and commit**

Run: `cd apps/api && go test ./internal/invite ./internal/workspace ./internal/platform/http -run 'Test.*(Invite|Member|Router)' -count=1`

Expected: PASS.

Commit: `git add apps/api/internal/invite apps/api/internal/workspace apps/api/internal/platform/http apps/api/cmd/api/main.go && git commit -m "feat: expose workspace invitation API"`

### Task 4: Build invitation management and acceptance UI

**Files:**

- Create: `apps/web/src/lib/{invite-api.ts,invite-api.test.ts}`, `apps/web/src/app/invites/[token]/{page.tsx,page.test.tsx}`
- Modify: `apps/web/src/lib/{workspace-api.ts,workspace-api.test.ts}`, `apps/web/src/app/[workspaceSlug]/{page.tsx,page.test.tsx}`, `apps/web/src/app/{login/page.tsx,register/page.tsx}`, `apps/web/src/features/auth/auth-form.tsx`

**Interfaces:**

- `InviteApi` provides `create`, `list`, `revoke`, `preview`, and `accept` typed methods.
- `WorkspaceMember` gets nullable `addedByUserId`/`addedByName`; direct `WorkspaceApi.addMember` is removed.
- Login/register preserve only an app-relative `next` path beginning with `/`.

- [ ] **Step 1: Write failing frontend tests**

Test protected API construction for all invite methods. Test owner/admin invite form visibility and member/viewer absence; show “Added by” only to managers; show admin removal only if the target provenance equals the current admin. Test preview, success, expired/unavailable, and mismatch acceptance pages plus login/register continuation links.

```tsx
expect(await screen.findByRole("button", { name: "Send invitation" })).toBeTruthy();
expect(screen.getByText("Added by Admin One")).toBeTruthy();
expect(screen.queryByRole("button", { name: "Remove Peer Admin" })).toBeNull();
```

- [ ] **Step 2: Run tests red**

Run: `pnpm --filter @taskline/web test -- src/lib/invite-api.test.ts 'src/app/[workspaceSlug]/page.test.tsx' 'src/app/invites/[token]/page.test.tsx'`

Expected: FAIL because the client API, route, and provenance fields do not exist.

- [ ] **Step 3: Implement typed APIs and workspace controls**

Implement `InviteApi` through `ProtectedRequest`, encoding the opaque token only in the path. Replace direct addition with email/role invite form, active-invite list, and authorized revoke control. Display “Added by <name>” only to owner/admin viewers. Apply the exact UI removal hierarchy but retain API enforcement as authoritative.

```ts
create(workspaceId: string, input: { email: string; role: InviteRole }): Promise<WorkspaceInvite>
preview(token: string): Promise<InvitePreview>
accept(token: string): Promise<WorkspaceSummary>
```

- [ ] **Step 4: Implement the acceptance route and safe auth continuation**

The page loads safe preview data and never renders recipient email. It provides sign-in/register links with `next=/invites/<token>`, accepts only app-relative next values after authentication, submits acceptance, and redirects to the resulting workspace. Render generic unavailable copy for 404 and non-sensitive email-mismatch copy for 403.

- [ ] **Step 5: Verify and commit**

Run: `pnpm --filter @taskline/web test -- src/lib/invite-api.test.ts 'src/app/[workspaceSlug]/page.test.tsx' 'src/app/invites/[token]/page.test.tsx'`

Expected: PASS.

Commit: `git add apps/web/src/lib apps/web/src/app apps/web/src/features/auth/auth-form.tsx && git commit -m "feat: add workspace invitation UI"`

### Task 5: Document and verify the complete phase

**Files:**

- Modify: `README.md`, `apps/api/.env.example`, `.env.example`
- Verify: `apps/api/internal/platform/database/sqlc`, complete Go and frontend suites

- [ ] **Step 1: Document configuration and workflow**

Add safe placeholders for the five SMTP variables, require STARTTLS and a public `WEB_ORIGIN`, and replace direct member-add instructions with the email invitation workflow. Document 24-hour, single-use, exact-email acceptance and the owner/admin hierarchy.

- [ ] **Step 2: Regenerate and run backend checks**

Run: `cd apps/api && go tool sqlc generate && git diff --exit-code -- internal/platform/database/sqlc && gofmt -l . && go vet ./... && go test ./... && go build ./... && golangci-lint run ./...`

Expected: no formatting output; every check exits 0. Run repository integration tests with Compose PostgreSQL and `DATABASE_URL` configured.

- [ ] **Step 3: Run frontend checks**

Run: `pnpm --filter @taskline/web test && pnpm lint && pnpm typecheck && pnpm format:check && pnpm --filter @taskline/web exec next build --webpack`

Expected: every test, static check, formatter check, and production build passes.

- [ ] **Step 4: Perform delivery and browser smoke tests**

With safe SMTP credentials and a disposable recipient, create an invite, verify an opaque link arrives, reject it while logged in with another email, accept it with the invited email, verify the second attempt fails, and verify owner/admin view shows who added the new member.

- [ ] **Step 5: Inspect and commit**

Run: `git diff --check && git diff --stat && git status --short`

Expected: only scoped changes and no whitespace errors.

Commit: `git add README.md .env.example apps/api/.env.example docs/superpowers/plans/2026-09-09-workspace-invitations.md && git commit -m "docs: document workspace invitations"`
