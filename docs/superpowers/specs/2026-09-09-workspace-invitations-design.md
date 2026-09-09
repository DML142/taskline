# Workspace Invitations Design

## Goal

Let workspace owners and administrators invite people by email through a one-time, 24-hour link that can be accepted only by an account using the invited email address. Preserve an auditable record of who added each member and enforce the agreed administrator hierarchy when managing members.

## Scope and constraints

- Invitations use configured real SMTP, not a development-only link log or a third-party delivery provider.
- A link contains a high-entropy opaque secret only; it must not contain email addresses, workspace identifiers, roles, or derived invitation codes.
- The database stores only a SHA-256 hash of the secret. An invite is valid for 24 hours and can be accepted at most once.
- Accepted email comparison is case-insensitive after trimming surrounding whitespace and lowercasing.
- Owners can invite `ADMIN`, `MEMBER`, and `VIEWER` roles; they can cancel every invite and remove every member except themselves.
- Administrators can invite `ADMIN`, `MEMBER`, and `VIEWER` roles; they can remove `MEMBER` and `VIEWER` accounts and only those `ADMIN` accounts whose membership records name that administrator as `added_by_user_id`.
- An administrator cannot remove the owner, themselves, or an administrator added by anyone else. These conditions are authorization rules enforced in the Go service layer.
- Workspace membership lists show who added a member to workspace owners and administrators. Existing memberships without provenance display an unknown/legacy value and do not grant administrator-to-administrator removal rights.

## Data model

Migration `000006_workspace_invites` adds:

1. Nullable `added_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL` to `workspace_members`, plus an index supporting administrator hierarchy checks. The creator's original owner membership and all pre-existing memberships remain `NULL`.
2. `workspace_invites`, containing `id UUID`, `workspace_id`, normalized `email`, target `role`, `token_hash BYTEA UNIQUE`, `created_by_user_id`, `expires_at`, `accepted_at`, `revoked_at`, and timestamps. Foreign keys cascade with workspace deletion; actor references use `ON DELETE SET NULL` only where preserving historic invite audit data remains useful.
3. A partial unique index ensuring one unconsumed, unrevoked invitation per normalized `(workspace_id, email)`. Before creating a replacement, the service revokes both expired and active rows for that email in the same transaction; the index intentionally does not call `now()`, which PostgreSQL does not allow in an index predicate.

When an invite is accepted, the service transaction locks the invite, rechecks expiry/revocation/acceptance, confirms the authenticated account email, creates `workspace_members` with `added_by_user_id = invite.created_by_user_id`, and sets `accepted_at`. A failed email comparison must not consume the invitation.

## API and authorization

The existing direct `POST /api/v1/workspaces/{workspaceID}/members` endpoint is removed from the web workflow and replaced with these authenticated endpoints:

- `POST /api/v1/workspaces/{workspaceID}/invites`: validates `{ email, role }`, applies owner/admin authorization, replaces any active invitation for the email, and sends the invite email.
- `GET /api/v1/workspaces/{workspaceID}/invites`: returns active invitations only to owners and administrators. An admin sees only invitations they created; an owner sees all.
- `DELETE /api/v1/workspaces/{workspaceID}/invites/{inviteID}`: owner can revoke any active invite; an admin can revoke only their own.
- `GET /api/v1/workspace-invites/{token}`: validates the opaque token and returns only the workspace display name, requested role, and expiry needed to render the acceptance screen.
- `POST /api/v1/workspace-invites/{token}/accept`: requires bearer authentication, accepts the token only when the authenticated account email equals the invited email, and returns the resulting workspace membership.

`GET /api/v1/workspaces/{workspaceID}/members` extends every member response with optional `addedByUserId` and `addedByName`. The existing member deletion route uses a shared `canRemoveMember` service policy covering owner and admin hierarchy. Role changes remain owner-only in this phase; changing an existing member into an administrator does not fabricate provenance and remains outside the new admin-to-admin removal path.

Invalid, expired, consumed, revoked, or unknown secrets yield the same generic not-found response. The acceptance screen may show generic expired/unavailable copy but never the invite email address. Email mismatch returns `403` and does not disclose the invited address. Input errors return `400`, authorisation failures `403`, and SMTP delivery failures `503` with a generic retryable error.

## Mail delivery

`internal/invite` owns the invite service, repository, HTTP handler, token creation, and an `InviteMailer` interface. An SMTP adapter configures host, port, username, password, sender, and explicit TLS/STARTTLS behavior from application configuration. `WEB_ORIGIN` forms an absolute acceptance URL: `${WEB_ORIGIN}/invites/${token}`.

Creating an invitation is intentionally durable only when delivery succeeds: persist a pending invite, send the message, then commit it. If SMTP fails, the transaction rolls back so users do not see an active but undelivered invitation. SMTP credentials never enter API responses or logs.

Required API environment variables are `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, and `SMTP_FROM`. Startup fails with a clear configuration error when any are missing or invalid. The README and API example environment file document them without real secrets.

## Web experience

The workspace page replaces the existing direct-add-member form with an owner/admin invite form (email and role). It shows active invitations, their expiration time, creator, and a revoke action only when allowed. The members list keeps the existing role and removal controls and adds “Added by <name>” to members for owners/admins; controls render only when the same hierarchy permits the action.

`/invites/[token]` is a client route. It loads the safe invite preview, explains the target workspace and role, then provides sign-in/registration paths that preserve the acceptance URL. Once authenticated, it submits acceptance; success routes to the workspace, while email mismatch, expiry, cancellation, and previous use receive concise non-sensitive error messages.

## Verification

Service tests cover token entropy/hash lookup, normalized email comparison, 24-hour expiry, one-time acceptance, cancellation, replacement, SMTP rollback, and complete owner/admin/member/viewer authorization behavior. Repository tests cover locking/transactional acceptance and membership provenance. HTTP tests assert response codes and generic responses that do not leak emails or tokens.

Frontend tests cover typed invite requests, owner/admin invite controls, visibility and revocation filtering, “Added by” display, permitted removal controls, invite preview, authentication continuation, acceptance success, and mismatch/expired states. Before completion run Go formatting, sqlc generation and diff, unit tests, vet, build, lint; then run frontend tests, lint, typecheck, formatting checks, and production build.
