-- 018_add_country_to_movies.down.sql

ALTER TABLE movies DROP COLUMN IF EXISTS country;
