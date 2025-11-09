.PHONY: run build docker-build docker-run migrate test

run:
	go run ./cmd/customer-service

build:
	go build -o bin/customer-service ./cmd/customer-service

docker-build:
	docker build -t customer-service:latest .

docker-run:
	@if [ ! -f .env ]; then \
	echo "Missing .env. Run: cp .env.example .env and edit credentials."; \
	exit 1; \
	fi
	docker run --rm -p 8080:8080 --env-file .env customer-service:latest

migrate:
	@if [ ! -f .env ]; then \
	echo "Missing .env. Run: cp .env.example .env and edit credentials."; \
	exit 1; \
	fi
	docker compose run --rm migrate

test:
	go test ./...
