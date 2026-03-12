-- 003_add_auth_to_profiles.down.sql

ALTER TABLE profiles DROP COLUMN IF EXISTS password_hash;
