-- 002_add_password_hash_to_profiles.up.sql
-- Добавляем хранение хэша пароля в таблицу профилей

ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';
