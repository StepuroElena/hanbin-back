-- 021_create_password_reset_tokens.up.sql
-- Токены для восстановления пароля («Забыли пароль?») — одноразовые, с ограниченным сроком жизни (1 час).

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id         BIGSERIAL    PRIMARY KEY,
    profile_id BIGINT       NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    token      VARCHAR(64)  NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT password_reset_tokens_token_unique UNIQUE (token)
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens (token);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_profile_id ON password_reset_tokens (profile_id);
