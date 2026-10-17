up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f

migrate-up:
	docker compose up migrate

migrate-down:
	docker compose run --rm migrate -path=/migrations -database "$${DATABASE_URL}" down 1

test:
	cd backend && go test ./...
