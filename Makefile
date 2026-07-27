.PHONY: run build tidy deps migrate-up migrate-down test

ifneq (,$(wildcard .env))
  include .env
  export
  DSN=$(DATABASE_URL)
else
  DSN=host=localhost port=5432 user=elenastepuro dbname=hanbin sslmode=disable
endif

DB_DSN ?= host=localhost port=5432 user=elenastepuro dbname=hanbin sslmode=disable

## run: запустить сервер локально
run:
	DATABASE_URL="$(DSN)" \
	ADDR=:8080 \
	ALLOWED_ORIGINS="http://localhost:3000,http://localhost:3001,http://localhost:5500,http://127.0.0.1:3000,http://127.0.0.1:5500" \
	go run ./cmd/api

## build: собрать бинарник
build:
	go build -o bin/hanbin-back ./cmd/api

## tidy: подтянуть зависимости
tidy:
	go mod tidy

## deps: скачать все зависимости
deps:
	go mod download

## test: запустить все тесты
test:
	go test ./...

## migrate-up: применить все миграции по порядку
migrate-up:
	psql "$(DSN)" -f migrations/001_create_profiles.up.sql
	psql "$(DSN)" -f migrations/002_create_dramas.up.sql
	psql "$(DSN)" -f migrations/003_add_auth_to_profiles.up.sql
	psql "$(DSN)" -f migrations/004_add_archive_fields_to_dramas.up.sql
	psql "$(DSN)" -f migrations/006_create_scrape_cache.up.sql
	psql "$(DSN)" -f migrations/007_add_voiceover_to_dramas.up.sql
	psql "$(DSN)" -f migrations/008_add_poster_url_to_dramas.up.sql
	psql "$(DSN)" -f migrations/009_add_source_url_to_dramas.up.sql
	psql "$(DSN)" -f migrations/010_create_streaming_sites.up.sql
	psql "$(DSN)" -f migrations/011_add_enabled_to_streaming_sites.up.sql
	psql "$(DSN)" -f migrations/012_remove_dorama24_streaming_sites.up.sql
	psql "$(DSN)" -f migrations/013_create_movies.up.sql
	psql "$(DSN)" -f migrations/014_add_watch_status_to_movies.up.sql
	psql "$(DSN)" -f migrations/015_add_genre_to_movies.up.sql
	psql "$(DSN)" -f migrations/016_add_archive_field_to_movies.up.sql
	psql "$(DSN)" -f migrations/017_expand_movie_watch_status.up.sql
	psql "$(DSN)" -f migrations/018_add_country_to_movies.up.sql

## migrate-down: откатить все миграции
migrate-down:
	psql "$(DSN)" -f migrations/018_add_country_to_movies.down.sql
	psql "$(DSN)" -f migrations/017_expand_movie_watch_status.down.sql
	psql "$(DSN)" -f migrations/016_add_archive_field_to_movies.down.sql
	psql "$(DSN)" -f migrations/015_add_genre_to_movies.down.sql
	psql "$(DSN)" -f migrations/014_add_watch_status_to_movies.down.sql
	psql "$(DSN)" -f migrations/013_create_movies.down.sql
	psql "$(DSN)" -f migrations/011_add_enabled_to_streaming_sites.down.sql
	psql "$(DSN)" -f migrations/010_create_streaming_sites.down.sql
	psql "$(DSN)" -f migrations/009_add_source_url_to_dramas.down.sql
	psql "$(DSN)" -f migrations/008_add_poster_url_to_dramas.down.sql
	psql "$(DSN)" -f migrations/007_add_voiceover_to_dramas.down.sql
	psql "$(DSN)" -f migrations/006_create_scrape_cache.down.sql
	psql "$(DSN)" -f migrations/004_add_archive_fields_to_dramas.down.sql
	psql "$(DSN)" -f migrations/003_add_auth_to_profiles.down.sql
	psql "$(DSN)" -f migrations/002_create_dramas.down.sql
	psql "$(DSN)" -f migrations/001_create_profiles.down.sql

## migrate-dramas-up: только дорамы
migrate-dramas-up:
	psql "$(DSN)" -f migrations/002_create_dramas.up.sql

## migrate-dramas-down: откат только дорам
migrate-dramas-down:
	psql "$(DSN)" -f migrations/002_create_dramas.down.sql

## migrate-auth-up: добавить auth-поля в profiles
migrate-auth-up:
	psql "$(DSN)" -f migrations/003_add_auth_to_profiles.up.sql

## migrate-auth-down: убрать auth-поля
migrate-auth-down:
	psql "$(DSN)" -f migrations/003_add_auth_to_profiles.down.sql

## migrate-archive-up: добавить поля архива/сезонов/прогресса
migrate-archive-up:
	psql "$(DSN)" -f migrations/004_add_archive_fields_to_dramas.up.sql

## migrate-archive-down: откатить поля архива/сезонов/прогресса
migrate-archive-down:
	psql "$(DSN)" -f migrations/004_add_archive_fields_to_dramas.down.sql

## migrate-scrape-cache-up: только таблица кеша скрейпинга
migrate-scrape-cache-up:
	psql "$(DSN)" -f migrations/006_create_scrape_cache.up.sql

## migrate-scrape-cache-down: откатить таблицу кеша скрейпинга
migrate-scrape-cache-down:
	psql "$(DSN)" -f migrations/006_create_scrape_cache.down.sql
