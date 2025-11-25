.PHONY: build run test migrate migrate-status migrate-down clean dev

build:
	docker-compose build

run:
	docker-compose up

test:
	go test ./...

migrate:
	goose -dir migrations postgres "postgres://postgres:password@localhost:5432/pr_reviewer?sslmode=disable" up

migrate-status:
	goose -dir migrations postgres "postgres://postgres:password@localhost:5432/pr_reviewer?sslmode=disable" status

migrate-down:
	goose -dir migrations postgres "postgres://postgres:password@localhost:5432/pr_reviewer?sslmode=disable" down

migrate-reset:
	goose -dir migrations postgres "postgres://postgres:password@localhost:5432/pr_reviewer?sslmode=disable" reset

clean:
	docker-compose down -v

dev:
	go run ./cmd/server