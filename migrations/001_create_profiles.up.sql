-- 001_create_profiles.up.sql
-- Таблица профилей пользователей

CREATE TABLE IF NOT EXISTS profiles (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT profiles_email_unique UNIQUE (email)
);

-- Индекс на email для быстрого поиска GetByEmail
CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles (email);
