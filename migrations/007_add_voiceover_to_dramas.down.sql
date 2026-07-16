-- 007_add_voiceover_to_dramas.down.sql

ALTER TABLE dramas
    DROP COLUMN IF EXISTS voiceover;
