-- 018_add_country_to_movies.up.sql
-- Страна выпуска фильма — опциональное поле (в отличие от жанра).

ALTER TABLE movies
    ADD COLUMN IF NOT EXISTS country VARCHAR(100) NOT NULL DEFAULT '';
