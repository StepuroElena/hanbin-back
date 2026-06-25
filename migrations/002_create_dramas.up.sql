-- 002_create_dramas.up.sql
-- Таблица дорам, привязанных к пользователям

DO $$ BEGIN
    CREATE TYPE drama_release_tag AS ENUM ('ongoing', 'released');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE drama_translation_tag AS ENUM ('translated', 'translating');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE drama_watch_status AS ENUM ('planned', 'watching', 'completed', 'dropped');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Дропаем старую таблицу и всё что от неё зависит (drama_tags и т.п.)
DROP TABLE IF EXISTS dramas CASCADE;

CREATE TABLE dramas (
    id               BIGSERIAL             PRIMARY KEY,
    profile_id       BIGINT                NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    title            VARCHAR(500)          NOT NULL,
    watch_url        TEXT                  NOT NULL,
    release_year     SMALLINT              NOT NULL CHECK (release_year BETWEEN 1900 AND 2100),
    release_tag      drama_release_tag     NOT NULL,
    translation_tag  drama_translation_tag NOT NULL,
    genre            VARCHAR(100)          NOT NULL,
    rating           NUMERIC(3,1)          CHECK (rating IS NULL OR (rating >= 0 AND rating <= 10)),
    watch_status     drama_watch_status    NOT NULL DEFAULT 'planned',
    country          VARCHAR(100)          NOT NULL,
    created_at       TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ           NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dramas_profile_id ON dramas (profile_id);
