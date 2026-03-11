-- 003_split_users_and_profiles.down.sql

DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS users;

-- Восстанавливаем исходную структуру profiles
CREATE TABLE IF NOT EXISTS profiles (
    id            BIGSERIAL    PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT profiles_email_unique UNIQUE (email)
);

CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles (email);
