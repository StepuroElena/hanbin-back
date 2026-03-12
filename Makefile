.PHONY: run build tidy deps migrate-up migrate-down test

DSN=host=localhost port=5432 user=elenastepuro dbname=hanbin sslmode=disable

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
	psql "$(DSN)" -f migrations/003_split_users_and_profiles.up.sql
	psql "$(DSN)" -f migrations/004_add_name_to_users.up.sql
	psql "$(DSN)" -f migrations/005_add_dramas_and_badges.up.sql

## migrate-down: откатить последнюю миграцию
migrate-down:
	psql "$(DSN)" -f migrations/003_split_users_and_profiles.down.sql
