.PHONY: build run test clean dev

build:
	docker-compose build

run:
	docker-compose up

test:
	go test ./...

clean:
	docker-compose down -v

dev:
	go run ./cmd/server