.PHONY: up down logs db-up db-down migrate-up migrate-down run test

up: db-up

down: db-down

logs:
	docker compose logs -f db

db-up:
	docker compose up -d db

db-down:
	docker compose down

migrate-up:
	set -a && . ./.env && set +a && migrate -path=backend/migrations -database "$$DATABASE_URL" up

migrate-down:
	set -a && . ./.env && set +a && migrate -path=backend/migrations -database "$$DATABASE_URL" down 1

run:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/server

test:
	cd backend && go test ./...
