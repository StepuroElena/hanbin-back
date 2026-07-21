-- 011_add_enabled_to_streaming_sites.down.sql

ALTER TABLE streaming_sites
    DROP COLUMN IF EXISTS enabled;
