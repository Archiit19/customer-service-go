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

## Continuous deployment to EC2 + AWS RDS

Pushing to the `main` branch triggers the GitHub Actions workflow defined in
`.github/workflows/deploy.yml`. The job ensures the EC2 instance always runs the
latest service image against your RDS instance:

1. Build the Docker image and bundle the SQL files from `migrations/`.
2. Copy the image tarball and migrations archive to the EC2 host.
3. Run the migrations from inside a temporary `postgres:16-alpine` container
   (`docker run --rm --network host ...`) so the SQL executes directly against
   RDS using the credentials stored in repository secrets.
4. Restart the long-running `customer-service` container with the new image and
   environment configured for RDS (`DB_HOST`, `DB_PORT`, etc.).

To confirm the job executed successfully, open the "Deploy to EC2 with Docker"
workflow run in GitHub Actions and verify the `Archive SQL migrations`, `Copy
image to EC2`, and `SSH into EC2 and deploy container` steps are green.

### Verifying RDS connectivity from EC2

Once the workflow finishes, log into the EC2 host:

```bash
ssh -i /path/to/key.pem ${EC2_USER}@${EC2_HOST}
```

Run a quick connectivity check against RDS using the same Docker image the
workflow used for migrations (set the `DB_*` environment variables first or
substitute literal values in the command):

```bash
docker run --rm --network host postgres:16-alpine \
  sh -c "psql \"host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME sslmode=$DB_SSLMODE\" -c 'SELECT 1'"
```

Next, confirm the service container is running the latest image and can reach
the database:

```bash
docker ps --filter "name=customer-service"
docker logs customer-service --tail=100
curl http://localhost:8080/healthz
```

If the connectivity check fails, ensure the RDS security group allows inbound
traffic from the EC2 instance's security group or private IP address, and that
the subnet routing permits communication.
