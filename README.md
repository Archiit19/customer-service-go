# Customer Service (Go + PostgreSQL on AWS RDS)

A production-ready Go 1.24 microservice that manages customer profiles and their KYC verification. The service exposes an HTTP API via `chi`, persists data in PostgreSQL (RDS compatible), and publishes an OpenAPI contract (`openapi.yaml`) for downstream consumers.

## Configuration
All configuration is environment-driven. Copy `.env.example` to `.env` and adjust as needed.

| Variable | Description | Default |
| --- | --- | --- |
| `APP_PORT` | HTTP listener port | `8080` |
| `LOG_LEVEL` | `DEBUG`, `INFO`, `WARN`, `ERROR` | `INFO` |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | PostgreSQL connectivity settings | `localhost:5432`, `postgres`, `postgres`, `customerdb` |
| `DB_SSLMODE` | `disable`, `require`, `verify-full` | `disable` (set to `require` for RDS) |
| `DB_MAX_CONNS` / `DB_MIN_CONNS` / `DB_MAX_IDLE_TIME` | pgx pool tuning knobs | `10`, `2`, `30m` |

For AWS RDS use `DB_SSLMODE=require` (or `verify-full` with your CA bundle).

## Database migration
Run the schema migration once per environment:

```bash
psql "host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME sslmode=$DB_SSLMODE" \
  -f migrations/0001_create_customers.sql
```

Without a local `psql` binary run the same migration through the official image:

```bash
docker run --rm --network host -v "$PWD/migrations:/migrations" postgres:16-alpine \
  psql "host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME sslmode=$DB_SSLMODE" \
  -f /migrations/0001_create_customers.sql
```

`docker compose run --rm migrate` targets the same script using the compose-defined Postgres container.

## Build, test, and run

```bash
go mod tidy
go test ./...
go run ./cmd/customer-service
```

Container workflow:

```bash
docker build -t customer-service:latest .
docker run --rm -p 8080:8080 --env-file .env customer-service:latest
```

Makefile shortcuts (`make run`, `make build`, `make docker-build`, `make docker-run`, `make migrate`) wrap the standard Go, Docker, and migration commands.

### Local Docker Compose stack

```bash
cp .env.example .env
docker compose up -d db
psql "host=localhost port=5432 user=postgres password=postgres dbname=customerdb sslmode=disable" \
  -f migrations/0001_create_customers.sql
go run ./cmd/customer-service
```

## API surface
- Base URL: `http://localhost:8080`
- REST resources under `/v1/customers`
  - `POST /v1/customers` – create customer profile
  - `GET /v1/customers?page&limit` – paginated listing with HATEOAS links for verification
  - `GET /v1/customers/{id}` – hydrated customer + verification metadata
  - `PATCH /v1/customers/{id}` – partial updates (name/email/phone)
  - `DELETE /v1/customers/{id}` – soft delete
  - `GET /v1/customers/{id}/status` – current verification record
  - `PATCH /v1/customers/{id}/verification` – create PAN entry or transition verification state
- Health: `GET /healthz` returns `{"status":"ok"}`

Refer to `openapi.yaml` for schemas, error models, and response codes. Regenerate client SDKs or documentation from this file as needed.

## Deployment on Minikube
Manifests live under `deploy/minikube` and provision PostgreSQL, the API deployment, Promtail/Loki logging stack, and the ingress gateway. Requirements: `minikube`, `kubectl`, and `docker`.

1. Start the cluster and ingress controller:
   ```bash
   minikube start
   minikube addons enable ingress
   ```
2. Build inside the Minikube Docker daemon:
   ```bash
   eval "$(minikube -p minikube docker-env)"
   docker build -t customer-service:latest .
   eval "$(minikube docker-env -u)"
   ```
3. Apply the manifests:
   ```bash
   kubectl apply -f deploy/minikube
   ```
4. Run the migration via port-forward:
   ```bash
   kubectl port-forward svc/postgres -n customer-service 5432:5432
   psql "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=customerdb sslmode=disable" \
     -f migrations/0001_create_customers.sql
   ```
5. Patch the ingress host and open a tunnel:
   ```bash
   MINIKUBE_IP=$(minikube ip)
   PUBLIC_HOST="customer-service.${MINIKUBE_IP}.nip.io"
   kubectl patch ingress customer-service -n customer-service \
     --type=json \
     -p="[{\"op\":\"replace\",\"path\":\"/spec/rules/0/host\",\"value\":\"${PUBLIC_HOST}\"}]"
   minikube tunnel
   ```
6. Verify the deployment:
   ```bash
   curl "http://${PUBLIC_HOST}/healthz"
   curl "http://${PUBLIC_HOST}/v1/customers"
   ```

   Replace `PUBLIC_HOST` with the value you set in the previous step if you are
   running the commands in a new shell.
