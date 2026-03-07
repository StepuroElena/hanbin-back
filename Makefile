.PHONY: run build tidy migrate-up migrate-down test

# Загружаем переменные из .env если файл существует
ifneq (,$(wildcard .env))
  include .env
  export
endif

## run: запустить сервер локально
run:
	DATABASE_URL="host=localhost port=5432 user=elenastepuro dbname=hanbin sslmode=disable" go run ./cmd/api

## build: собрать бинарник
build:
	go build -o bin/hanbin-back ./cmd/api

## tidy: подтянуть зависимости
tidy:
	go mod tidy

## test: запустить все тесты
test:
	go test ./...

## migrate-up: применить миграции
migrate-up:
	psql "host=localhost port=5432 user=elenastepuro dbname=hanbin sslmode=disable" -f migrations/001_create_profiles.up.sql

## migrate-down: откатить миграции
migrate-down:
	psql "host=localhost port=5432 user=elenastepuro dbname=hanbin sslmode=disable" -f migrations/001_create_profiles.down.sql
