# Multiuser Cluster Manager

Manage a Kubernetes cluster on behalf of multiple users.

## Local development
- Run `scripts/dev.sh` before committing; it executes `go fmt`, `go vet`, and `go test -race` across the repo.
- `go run ./cmd/mcm` starts the API on `:8080`; hit `/hello`, `/livez`, or `/readyz` to verify handlers in `internal/transport/http`.

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

