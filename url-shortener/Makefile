.PHONY: up down logs migrate test lint tidy run

up:
	docker compose up --build -d

down:
	docker compose down -v

logs:
	docker compose logs -f api

migrate:
	migrate -path migrations -database "$(POSTGRES_DSN)" up

test:
	go test ./... -race -count=1

lint:
	golangci-lint run

tidy:
	go mod tidy

run:
	go run ./cmd/api
