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

## Continuous deployment to Kubernetes

Pushing to the `main` branch triggers the GitHub Actions workflow in
`.github/workflows/deploy.yml`. The pipeline builds and publishes a container
image to GitHub Container Registry (GHCR), applies the manifests under
`deploy/minikube`, runs the SQL migrations against the in-cluster PostgreSQL
instance, and waits for the rollout to complete.

### Required repository secrets

| Secret | Purpose |
| ------ | ------- |
| `KUBE_CONFIG` | Base64-encoded kubeconfig file with permissions to deploy into the cluster. |
| `GHCR_USERNAME` / `GHCR_TOKEN` | Credentials used to create the `ghcr-credentials` image pull secret inside the cluster. The token must have `read:packages` and `write:packages` scopes. |
| `DB_USER` / `DB_PASSWORD` | Database credentials that are stored in the `customer-service-db-secret` secret and shared between PostgreSQL and the API deployment. |

The workflow also relies on the built-in `GITHUB_TOKEN` to push the image to
GHCR under the `ghcr.io/<owner>/<repo>` namespace.

### What the workflow does

1. Builds the service image with Docker Buildx and pushes it to GHCR tagged with
   the commit SHA and branch name.
2. Configures access to the Kubernetes API using the provided kubeconfig.
3. Applies the namespace, config map, PostgreSQL deployment/service, customer
   service deployment, and ingress manifests from `deploy/minikube`.
4. Recreates the database credentials secret and GHCR image pull secret so that
   credentials can be rotated without manual intervention.
5. Waits for PostgreSQL to become ready, copies the SQL files from `migrations/`
   into the pod, and executes them with `psql`.
6. Updates the `customer-service` deployment to use the freshly built image and
   waits for a successful rollout before printing the pod/service/ingress
   summary.

To verify a deployment, open the "Deploy to Kubernetes" workflow run in GitHub
Actions and confirm that the `Run database migrations` and `Wait for service
rollout` steps complete successfully.
