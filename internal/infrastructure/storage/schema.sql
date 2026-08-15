CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_invited_by
    ON users (invited_by);

CREATE UNIQUE INDEX idx_users_single_root
    ON users ((invited_by IS NULL))
    WHERE invited_by IS NULL;


CREATE TABLE credentials (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id  BYTEA NOT NULL,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key   BYTEA NOT NULL,
    sign_count   BIGINT NOT NULL DEFAULT 0,
    transports   TEXT[] NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_credentials_external_id
    ON credentials (external_id);

CREATE INDEX idx_credentials_user_id
    ON credentials (user_id);


CREATE TABLE invites (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,

    -- Audit information: who redeemed the invite.
    used_by    UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Actual state: whether the invite was redeemed.
    used_at    TIMESTAMPTZ,

    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_invites_code_hash
    ON invites (code_hash);

CREATE INDEX idx_invites_created_by_active
    ON invites (created_by, expires_at)
    WHERE used_at IS NULL;


CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,
    device_info TEXT NOT NULL DEFAULT '',
    ip_address  INET NOT NULL DEFAULT '0.0.0.0',
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_refresh_tokens_token_hash
    ON refresh_tokens (token_hash);

CREATE INDEX idx_refresh_tokens_user_active
    ON refresh_tokens (user_id, expires_at)
    WHERE revoked_at IS NULL;


CREATE TYPE wrapped_key_scope AS ENUM ('main', 'decoy');

CREATE TABLE wrapped_keys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    scope         wrapped_key_scope NOT NULL,
    wrapped_dek   BYTEA NOT NULL,
    wrap_algorithm TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_wrapped_keys_user_id
    ON wrapped_keys (user_id);

CREATE INDEX idx_wrapped_keys_credential_id
    ON wrapped_keys (credential_id);

CREATE UNIQUE INDEX idx_wrapped_keys_credential_scope
    ON wrapped_keys (credential_id, scope);