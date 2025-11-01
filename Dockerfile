# syntax=docker/dockerfile:1

FROM golang:1.23 AS builder
WORKDIR /src
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/customer-service ./cmd/customer-service

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /out/customer-service /app/customer-service
COPY .env.example /app/.env.example
EXPOSE 8080
ENTRYPOINT ["/app/customer-service"]
