-- 003_split_users_and_profiles.up.sql
-- Разделяем auth-данные (users) и данные профиля (profiles).
-- users  — учётная запись: email + password_hash, создаётся при регистрации.
-- profiles — публичный профиль пользователя, привязан к user_id.

-- 1. Таблица пользователей (auth)
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL    PRIMARY KEY,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- 2. Пересоздаём profiles: убираем email/password_hash, добавляем user_id
DROP TABLE IF EXISTS profiles;

CREATE TABLE IF NOT EXISTS profiles (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT profiles_user_id_unique UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles (user_id);
