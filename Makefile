.PHONY: compose-up run test sqlc web-install web-dev web-build tidy

ifneq (,$(wildcard .env))
include .env
export
endif

compose-up:
	docker compose up -d
	@echo "waiting for postgres..."
	@until docker compose exec -T postgres pg_isready -U bank -d bank >/dev/null 2>&1; do sleep 1; done
	@echo "postgres is ready"

run:
	go run ./cmd/server

test:
	go test ./...

sqlc:
	sqlc generate

web-install:
	cd web && npm install

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

tidy:
	go mod tidy
