-- 002_add_password_hash_to_profiles.down.sql

ALTER TABLE profiles
    DROP COLUMN IF EXISTS password_hash;
