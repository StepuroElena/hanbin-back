-- 014_add_watch_status_to_movies.down.sql

ALTER TABLE movies DROP COLUMN IF EXISTS watch_status;
