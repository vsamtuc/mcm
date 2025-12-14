
# Layout 

```
deploy/kustomize/mcm/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── mcm-deployment.yaml
│   ├── mcm-service.yaml
│   ├── mcm-ingress.yaml
│   ├── postgres-statefulset.yaml
│   ├── postgres-service.yaml
│   ├── postgres-service-headless.yaml
│   ├── keycloak-deployment.yaml
│   ├── keycloak-service.yaml
│   └── initdb-configmap.yaml         # creates DBs/users at first boot
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   ├── patches/
    │   │   ├── mcm-deployment.yaml   # debug env, replicas, resources
    │   │   ├── ingress-host.yaml     # host/tls
    │   │   └── keycloak-env.yaml     # “start-dev” or DB URL
    │   └── secret-generator.yaml     # non-prod only (optional)
    └── prod/
        ├── kustomization.yaml
        └── patches/
            ├── mcm-deployment.yaml
            ├── ingress-host.yaml
            └── storage-and-resources.yaml
```

---

# ▶️ Apply (examples)

```bash
# Dev
kubectl kustomize deploy/kustomize/mcm/overlays/dev | kubectl apply -f -

# Prod (expects secrets to already exist; no secretGenerator)
kubectl kustomize deploy/kustomize/mcm/overlays/prod | kubectl apply -f -
```

---

## ✅ Notes & tips

* **Names & DNS**: with `namePrefix: mcm-`, your services become `mcm-api`, `mcm-postgres`, `mcm-keycloak`. DNS: `mcm-api.mcm.svc.cluster.local`.
* **Keycloak vs DB**: the base uses Postgres; you can flip to `start-dev` (embedded H2) in dev overlay for speed.
* **Secrets**: keep out of git. For prod, create `postgres-superuser`, `mcm-db-secret`, `keycloak-db-secret`, `keycloak-admin` out-of-band (or use SOPS/ESO).
* **Ingress**: adjust `ingressClassName`/annotations for your ingress controller; add TLS in prod.
* **Health**: the probes shown are sane defaults; tune to your images/boot-time.
* **GitHub Actions**: set `GIT_SHA` and run `kubectl kustomize overlays/prod | kubectl apply -f -`.

---

## 🗄️ Schema migrations via Job

Migrations live in `db/migrations` and are executed with the `mcm-migrate` job found under `base/migrate-job`:

```bash
# Build/push the image referenced by the overlay (defaults: docker.io/vsam/mcm-migrate:dev)
make migrate-build

# Dev cluster
kubectl delete job/mcm-migrate -n mcm --ignore-not-found
kubectl kustomize deploy/kustomize/base/migrate-job/overlays/dev | kubectl apply -n mcm -f -
kubectl wait --for=condition=complete -n mcm job/mcm-migrate

# Prod (image name/tag overridden via ${GIT_SHA})
kubectl kustomize deploy/kustomize/base/migrate-job/overlays/prod | kubectl apply -n mcm -f -
```

Running the job is idempotent; reapply after deleting any finished job to rerun migrations during a rollout. The primary dev/prod overlays already include the migrate job as a resource, so standard `make dev-apply` / `make prod-apply` will automatically create it before deploying the API pods.

If you paste your **image name**, **desired ingress host**, and whether Keycloak should be **dev** or **DB-backed** in **dev**, I’ll tailor the overlays precisely and trim any extras.

