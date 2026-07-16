-- 008_add_poster_url_to_dramas.down.sql

ALTER TABLE dramas
    DROP COLUMN IF EXISTS poster_url;
