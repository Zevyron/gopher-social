include .envrc
DB_URL ?= postgres://admin:adminpassword@localhost:5432/social?sslmode=disable
MIGRATIONS_PATH=./cmd/migrate/migrations

.PHONY: migrate-create migrate-up migrate-down seed

migrate-create:
	@migrate create -seq -ext sql -dir $(MIGRATIONS_PATH) $(filter-out $@,$(MAKECMDGOALS))

migrate-up:
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

migrate-down:
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down $(filter-out $@,$(MAKECMDGOALS))

seed:
	@go run cmd/migrate/seed/main.go