.PHONY: up down build test lint run-api run-bot web-dev tidy

up:
	docker compose up --build -d

down:
	docker compose down

build:
	go build -o bin/api ./cmd/api
	go build -o bin/bot ./cmd/bot

test:
	go test ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

run-api:
	MIGRATIONS_DIR=migrations go run ./cmd/api

run-bot:
	go run ./cmd/bot

web-dev:
	cd web && npm run dev
