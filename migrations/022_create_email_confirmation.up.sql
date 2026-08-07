-- 022_create_email_confirmation.up.sql
-- Подтверждение email при регистрации: колонка email_confirmed_at в profiles + таблица
-- одноразовых токенов подтверждения (тот же паттерн, что и password_reset_tokens, но
-- живут дольше — 24 часа вместо 1 часа, см. service.Register).

ALTER TABLE profiles ADD COLUMN IF NOT EXISTS email_confirmed_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS email_confirmation_tokens (
    id         BIGSERIAL    PRIMARY KEY,
    profile_id BIGINT       NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    token      VARCHAR(64)  NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT email_confirmation_tokens_token_unique UNIQUE (token)
);

CREATE INDEX IF NOT EXISTS idx_email_confirmation_tokens_token ON email_confirmation_tokens (token);
CREATE INDEX IF NOT EXISTS idx_email_confirmation_tokens_profile_id ON email_confirmation_tokens (profile_id);
