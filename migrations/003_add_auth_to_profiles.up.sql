-- 003_add_auth_to_profiles.up.sql
-- Добавляем поля для аутентификации в таблицу профилей

ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';
