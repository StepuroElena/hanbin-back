-- 016_add_archive_field_to_movies.up.sql
-- Архивирование фильма — так же, как у дорам: is_archived, без физического удаления.

ALTER TABLE movies
    ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_movies_is_archived ON movies (is_archived);
