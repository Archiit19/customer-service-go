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


## Deploy to Minikube with an Ingress gateway

The repository ships with Kubernetes manifests under `deploy/minikube` that
spin up PostgreSQL, the customer service API, and an ingress gateway that
exposes the API publicly. The steps below assume that you have `minikube`,
`kubectl`, and `docker` installed locally.

1. **Start Minikube and enable the ingress controller**

   ```bash
   minikube start
   minikube addons enable ingress
   ```

2. **Build the service image inside the Minikube Docker daemon**

   ```bash
   eval "$(minikube -p minikube docker-env)"
   docker build -t customer-service:latest .
   ```

   > Tip: run `eval "$(minikube docker-env -u)"` afterwards to restore your
   > original Docker context.

3. **Apply the Kubernetes manifests**

   ```bash
   kubectl apply -f deploy/minikube
   ```

4. **Run the database migration**

   Forward PostgreSQL to your machine and apply the SQL migration using the
   bundled script:

   ```bash
   kubectl port-forward svc/postgres -n customer-service 5432:5432
   # in a new terminal
   psql "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=customerdb sslmode=disable" \
     -f migrations/0001_create_customers.sql
   ```

   After the migration succeeds, stop the `kubectl port-forward` command.

5. **Expose the ingress publicly**

   Pick a DNS-friendly host name that maps to your Minikube IP. The example
   below uses the free `nip.io` wildcard domain:

   ```bash
   MINIKUBE_IP=$(minikube ip)
   PUBLIC_HOST="customer-service.${MINIKUBE_IP}.nip.io"
   kubectl patch ingress customer-service -n customer-service \
     --type='json' \
     -p="[{\"op\":\"replace\",\"path\":\"/spec/rules/0/host\",\"value\":\"${PUBLIC_HOST}\"}]"
   ```

   Start the tunnel so that the `LoadBalancer` IP becomes reachable from the
   internet:

   ```bash
   minikube tunnel
   ```

6. **Verify the deployment**

   With the tunnel running you can reach the API through the ingress gateway:

   ```bash
   curl "http://${PUBLIC_HOST}/healthz"
   curl "http://${PUBLIC_HOST}/api/v1/customers"
   ```

   Replace `PUBLIC_HOST` with the value you set in the previous step if you are
   running the commands in a new shell.
