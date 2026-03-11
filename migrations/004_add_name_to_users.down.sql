-- 004_add_name_to_users.down.sql

ALTER TABLE users
    DROP COLUMN IF EXISTS name;
