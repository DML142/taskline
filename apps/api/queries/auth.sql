-- name: CreateUser :one
INSERT INTO users (email, name, password_hash)
VALUES ($1, $2, $3)
RETURNING id, email, name, password_hash, created_at, updated_at, email_verified_at;

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, created_at, updated_at, email_verified_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, name, password_hash, created_at, updated_at, email_verified_at
FROM users
WHERE id = $1;

-- name: HasActiveSession :one
SELECT EXISTS (
    SELECT 1 FROM sessions
    JOIN users ON users.id = sessions.user_id
    WHERE sessions.id = $1 AND sessions.user_id = $2 AND sessions.revoked_at IS NULL AND sessions.expires_at > now() AND users.email_verified_at IS NOT NULL
);

-- name: CreateSession :one
INSERT INTO sessions (id, user_id, family_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, family_id, token_hash, expires_at, revoked_at, replaced_by, created_at;

-- name: GetSessionByTokenHashForUpdate :one
SELECT id, user_id, family_id, token_hash, expires_at, revoked_at, replaced_by, created_at
FROM sessions
WHERE token_hash = $1
FOR UPDATE;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = now(), replaced_by = $2
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeSessionFamily :exec
UPDATE sessions
SET revoked_at = now()
WHERE family_id = $1 AND revoked_at IS NULL;

-- name: RevokeSessionByTokenHash :exec
UPDATE sessions
SET revoked_at = now()
WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: GetUserForSession :one
SELECT u.id, u.email, u.name, u.password_hash, u.created_at, u.updated_at, u.email_verified_at
FROM users AS u
JOIN sessions AS s ON s.user_id = u.id
WHERE s.id = $1;

-- name: DeleteOpenEmailVerificationTokens :exec
DELETE FROM email_verification_tokens
WHERE user_id = $1 AND used_at IS NULL;

-- name: CreateEmailVerificationToken :one
INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, used_at, created_at;

-- name: GetEmailVerificationTokenByHashForUpdate :one
SELECT id, user_id, token_hash, expires_at, used_at, created_at
FROM email_verification_tokens
WHERE token_hash = $1
FOR UPDATE;

-- name: UseEmailVerificationToken :execrows
UPDATE email_verification_tokens
SET used_at = now()
WHERE id = $1 AND used_at IS NULL;

-- name: VerifyUserEmail :execrows
UPDATE users
SET email_verified_at = now()
WHERE id = $1 AND email_verified_at IS NULL;
