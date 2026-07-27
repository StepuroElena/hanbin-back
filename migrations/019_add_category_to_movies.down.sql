-- 019_add_category_to_movies.down.sql

ALTER TABLE movies
    DROP COLUMN IF EXISTS category;
