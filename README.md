# Multiuser Cluster Manager

Manage a Kubernetes cluster on behalf of multiple users.

## Local development
- Run `scripts/dev.sh` before committing; it executes `go fmt`, `go vet`, and `go test -race` across the repo.
- `go run ./cmd/mcm` starts the API on `:8080`; hit `/hello`, `/livez`, or `/readyz` to verify handlers in `internal/transport/http`.
- Integration tests use Testcontainers (see `internal/app/app_integration_test.go`); keep Docker running locally or pass `-short` to skip them.
- Experimental JSON endpoints under `/api/courses` expose CRUD over the in-memory store defined in `pkg/course` + `internal/course/memory`.

## Database migrations
- SQL migrations live in `db/migrations`; `0001_init_schema` seeds students, courses, teams, and approval tables.
- Build the migration image (bundles the `migrate` CLI + SQL) with `make migrate-build`. Override `MIGRATE_IMAGE` to push to your registry.
- The main dev/prod overlays already include the `mcm-migrate` job, so every `make dev-apply` / `make prod-apply` run will create the job, execute pending migrations, and clean it up via `ttlSecondsAfterFinished`.
- Use the standalone targets when you need to run migrations outside a deploy:
	- `make dev-migrate` → applies `deploy/kustomize/base/migrate-job/overlays/dev` after deleting any old job and waits for completion.
	- `make prod-migrate` → same flow but points at the prod overlay (uses `${GIT_SHA}` for image tags).

## Deploying
- `make dev-apply` / `make prod-apply` wrap the main Kustomize overlays under `deploy/kustomize/overlays` (which now include the migrate job resource).
- The `mcm-migrate` job fires automatically on each apply; rerun manually via the dedicated targets if you need to verify schema changes without touching the API Deployment.

## Keycloak realm
- Import `deploy/kustomize/base/keycloak-realm.json` into your cluster Keycloak before exercising authenticated endpoints. The CLI bundled with the Keycloak pod works well:
	- `kubectl -n mcm exec deploy/mcm-keycloak -- /opt/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080 --realm master --user $KEYCLOAK_ADMIN --password $KEYCLOAK_ADMIN_PASSWORD`
	- `kubectl -n mcm exec deploy/mcm-keycloak -- /opt/keycloak/bin/kcadm.sh create realms -f /opt/keycloak/data/import/keycloak-realm.json`
- The realm now seeds a default `mcm-admin` user with the `admin` realm role and a temporary password (`ChangeMe123!`). Log into Keycloak after the import and set a permanent password before sharing the account.
- When you tweak the realm JSON, rerun the second command to re-import so that the `profile`, `email`, and `roles` scopes continue to show up in the JWTs consumed by the API.
- Override `KEYCLOAK_URL`, `KEYCLOAK_REALM`, `KEYCLOAK_CLIENT_ID`, and (optionally) `AUTH_SKIP_PATHS` via the dev/prod deployment patches to match your cluster DNS if you move Keycloak behind an ingress.

