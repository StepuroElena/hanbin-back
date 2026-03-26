-- 004_add_archive_fields_to_dramas.down.sql

ALTER TABLE dramas
    DROP COLUMN IF EXISTS is_archived,
    DROP COLUMN IF EXISTS episode_duration_min,
    DROP COLUMN IF EXISTS seasons,
    DROP COLUMN IF EXISTS progress;
