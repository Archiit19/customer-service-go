# Customer Service (Go + PostgreSQL on AWS RDS)

A minimal, production-ready Go microservice that manages customer profiles and KYC status.
Uses PostgreSQL (AWS RDS compatible) via `pgx` pool; router is `chi`.

## Features
- CRUD for customers
- PATCH for KYC status
- Soft delete (via `deleted_at`)
- Partial unique indexes on email/phone (ignore soft-deleted rows)
- Clean layering: handlers → service → repository → PostgreSQL
- Simple SQL migrations
- Dockerfile + Makefile
- OpenAPI spec

## Quickstart

### 1) Configure environment
Copy `.env.example` to `.env` and set values.

For **AWS RDS**, keep `DB_SSLMODE=require` (or `verify-full` with proper certs). Example:
```env
DB_HOST=mydb.abcdefg1234.us-east-1.rds.amazonaws.com
DB_PORT=5432
DB_USER=appuser
DB_PASSWORD=supersecret
DB_NAME=customerdb
DB_SSLMODE=require
```

### 2) Run migrations
Use psql or any DB client to apply `migrations/0001_create_customers.sql`:

```bash
psql "host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME sslmode=$DB_SSLMODE"       -f migrations/0001_create_customers.sql
```

> Tip: If you don't have `psql`, you can use the official postgres Docker image:
```bash
docker run --rm -it --network host -v "$PWD/migrations:/migrations" postgres:16-alpine       psql "host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME sslmode=$DB_SSLMODE"       -f /migrations/0001_create_customers.sql
```

### 3) Build & run
```bash
go mod tidy
go run ./cmd/customer-service
# or with Docker
docker build -t customer-service:latest .
docker run --rm -p 8080:8080 --env-file .env customer-service:latest
```

### Running migrations with Docker Compose
```bash
docker compose run --rm migrate
```

### Local development (optional)
If you want to run Postgres locally instead of AWS:
```bash
cp .env.example .env       # make sure DB_HOST=db, DB_SSLMODE=disable for the local container
docker compose up -d db
export $(cat .env.local | xargs)  # or fill your environment manually
psql "host=localhost port=5432 user=postgres password=postgres dbname=customerdb sslmode=disable"       -f migrations/0001_create_customers.sql
go run ./cmd/customer-service
```

## API (Base Path: `/api/v1`)
- `POST /customers`
- `GET /customers?kyc_status=VERIFIED&page=1&limit=20`
- `GET /customers/{id}`
- `PUT /customers/{id}`
- `PATCH /customers/{id}/kyc`
- `DELETE /customers/{id}`

See `openapi.yaml` for the full contract.

## Health check
- `GET /healthz` → `{ "status": "ok" }`

## Makefile (shortcuts)
- `make run` – run the service
- `make build` – build the service
- `make docker-build` – build docker image
- `make docker-run` – run via docker
- `make migrate` – apply SQL migrations using the compose helper