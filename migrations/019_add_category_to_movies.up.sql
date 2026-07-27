-- 019_add_category_to_movies.up.sql
-- Категория фильма — опциональное поле (в отличие от жанра). Значение выбирается
-- из персонального списка категорий пользователя (см. 020_create_movie_categories),
-- но хранится здесь просто строкой — так же, как genre/country.

ALTER TABLE movies
    ADD COLUMN IF NOT EXISTS category VARCHAR(100) NOT NULL DEFAULT '';
