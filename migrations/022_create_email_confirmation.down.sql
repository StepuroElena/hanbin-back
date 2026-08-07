-- 022_create_email_confirmation.down.sql

DROP TABLE IF EXISTS email_confirmation_tokens;
ALTER TABLE profiles DROP COLUMN IF EXISTS email_confirmed_at;
