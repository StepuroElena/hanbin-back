.PHONY: run build tidy deps migrate-up migrate-down test

ifneq (,$(wildcard .env))
  include .env
  export
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

<## migrate-up: применить все миграции по порядку
migrate-up:
psql "$(DSN)" -f migrations/001_create_profiles.up.sql
	psql "$(DSN)" -f migrations/002_create_dramas.up.sql
	psql "$(DSN)" -f migrations/003_add_auth_to_profiles.up.sql
	psql "$(DSN)" -f migrations/004_add_archive_fields_to_dramas.up.sql
## migrate-down: откатить все миграции
migrate-down:
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
