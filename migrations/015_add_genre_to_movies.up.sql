-- 015_add_genre_to_movies.up.sql
-- Жанр фильма — обязательное поле (в отличие от года, который опционален).

ALTER TABLE movies
    ADD COLUMN IF NOT EXISTS genre VARCHAR(100) NOT NULL DEFAULT '';
