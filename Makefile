BACKEND_DIR := backend
export DATABASE_URL ?= postgres://movies:movies@localhost:5432/movies?sslmode=disable

.PHONY: up down migrate status generate run test test-backend test-frontend lint-frontend build-frontend check-generated check

up:
	docker compose up -d --build postgres migrate api

down:
	docker compose down

migrate:
	cd $(BACKEND_DIR) && tern migrate --conn-string "$(DATABASE_URL)" --migrations migrations

status:
	cd $(BACKEND_DIR) && tern status --conn-string "$(DATABASE_URL)" --migrations migrations

generate:
	cd $(BACKEND_DIR) && sqlc generate

run:
	cd $(BACKEND_DIR) && go run ./cmd/api

test: test-backend

test-backend:
	cd $(BACKEND_DIR) && go test ./...

test-frontend:
	npm --prefix frontend test

lint-frontend:
	npm --prefix frontend run lint

build-frontend:
	npm --prefix frontend run build

check-generated: generate
	git diff --exit-code -- backend/internal/repo/postgres/gen

check: test-backend test-frontend lint-frontend build-frontend check-generated
