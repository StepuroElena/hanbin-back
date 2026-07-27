-- 015_add_genre_to_movies.down.sql

ALTER TABLE movies DROP COLUMN IF EXISTS genre;
