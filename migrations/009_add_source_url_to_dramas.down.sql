-- 009_add_source_url_to_dramas.down.sql

ALTER TABLE dramas
    DROP COLUMN IF EXISTS source_url;
