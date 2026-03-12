-- 004_add_name_to_users.up.sql
-- Имя пользователя хранится на уровне учётной записи (users),
-- а не только в профиле — нужно для регистрации.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT '';
