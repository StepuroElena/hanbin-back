.PHONY: run build tidy migrate-up migrate-down test

ifneq (,$(wildcard .env))
  include .env
  export
endif

DB_DSN ?= host=localhost port=5432 user=elenastepuro dbname=hanbin sslmode=disable

## run: запустить сервер локально
run:
	DATABASE_URL="$(DB_DSN)" go run ./cmd/api

## build: собрать бинарник
build:
	go build -o bin/hanbin-back ./cmd/api

## tidy: подтянуть зависимости
tidy:
	go mod tidy

## test: запустить все тесты
test:
	go test ./...

## migrate-up: применить все миграции
migrate-up:
	psql "$(DB_DSN)" -f migrations/001_create_profiles.up.sql
	psql "$(DB_DSN)" -f migrations/002_create_dramas.up.sql
	psql "$(DB_DSN)" -f migrations/003_add_auth_to_profiles.up.sql

## migrate-down: откатить все миграции
migrate-down:
	psql "$(DB_DSN)" -f migrations/003_add_auth_to_profiles.down.sql
	psql "$(DB_DSN)" -f migrations/002_create_dramas.down.sql
	psql "$(DB_DSN)" -f migrations/001_create_profiles.down.sql

## migrate-dramas-up: только дорамы
migrate-dramas-up:
	psql "$(DB_DSN)" -f migrations/002_create_dramas.up.sql

## migrate-dramas-down: откат только дорам
migrate-dramas-down:
	psql "$(DB_DSN)" -f migrations/002_create_dramas.down.sql

## migrate-auth-up: добавить auth-поля в profiles
migrate-auth-up:
	psql "$(DB_DSN)" -f migrations/003_add_auth_to_profiles.up.sql

## migrate-auth-down: убрать auth-поля
migrate-auth-down:
	psql "$(DB_DSN)" -f migrations/003_add_auth_to_profiles.down.sql
