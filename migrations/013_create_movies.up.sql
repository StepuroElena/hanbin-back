-- 013_create_movies.up.sql
-- Таблица фильмов, привязанных к пользователям (минимальная версия — просто список,
-- без статусов/жанров/стран/архива, как у дорам — это может появиться позже)

CREATE TABLE IF NOT EXISTS movies (
    id           BIGSERIAL     PRIMARY KEY,
    profile_id   BIGINT        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    title        VARCHAR(500)  NOT NULL,
    release_year SMALLINT      CHECK (release_year IS NULL OR (release_year BETWEEN 1900 AND 2100)),
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_movies_profile_id ON movies (profile_id);
