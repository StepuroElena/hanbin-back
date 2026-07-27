-- 016_add_archive_field_to_movies.down.sql

DROP INDEX IF EXISTS idx_movies_is_archived;
ALTER TABLE movies DROP COLUMN IF EXISTS is_archived;
