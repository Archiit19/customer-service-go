.PHONY: run build docker-build docker-run lint test

    run:
	go run ./cmd/customer-service

    build:
	go build -o bin/customer-service ./cmd/customer-service

    docker-build:
	docker build -t customer-service:latest .

    docker-run:
	docker run --rm -p 8080:8080 --env-file .env.local customer-service:latest

    test:
	go test ./...
