KUSTOMIZE ?= kubectl kustomize
DOCKER ?= docker
DOCKERHUB_USER ?= vsam
K8S_NS ?= mcm

K8S_CONTEXT ?= mcm

MIGRATE_DOCKERFILE ?= build/docker/Dockerfile.migrate
MIGRATE_IMAGE ?= $(DOCKERHUB_USER)/mcm-migrate:dev
MIGRATE_OVERLAY_DEV ?= deploy/kustomize/jobs/migrate/overlays/dev
MIGRATE_OVERLAY_PROD ?= deploy/kustomize/jobs/migrate/overlays/prod

.PHONY: dev-build dev-apply dev-diff prod-apply prod-diff migrate-build dev-migrate prod-migrate

all:
	@echo "Please specify a target. Available targets are:"
	@echo dev-build
	@echo dev-apply
	@echo dev-diff
	@echo prod-apply
	@echo migrate-build
	@echo dev-migrate
	@echo prod-migrate

minikube-apply:
	minikube image load $(DOCKERHUB_USER)/mcm:dev
	minikube image load $(MIGRATE_IMAGE)
	kubectl apply -k deploy/kustomize/overlays/dev

minikube-migrate:
	kubectl delete job/mcm-migrate -n $(K8S_NS) --ignore-not-found
	kubectl apply -k $(MIGRATE_OVERLAY_DEV)
	kubectl wait --for=condition=complete --timeout=120s -n $(K8S_NS) job/mcm-mcm-migrate

minikube-delete:
	kubectl delete -k deploy/kustomize/overlays/dev

dev-build: migrate-build
	$(DOCKER) build -t $(DOCKERHUB_USER)/mcm:dev -f build/docker/Dockerfile .

dev-apply:
	$(KUSTOMIZE) deploy/kustomize/overlays/dev | kubectl --context=$(K8S_CONTEXT) apply -n $(K8S_NS) -f -

dev-diff:
	$(KUSTOMIZE) deploy/kustomize/overlays/dev | kubectl --context=$(K8S_CONTEXT) diff -n $(K8S_NS) -f -

prod-apply:
	$(KUSTOMIZE) deploy/kustomize/overlays/prod | kubectl --context=$(K8S_CONTEXT) apply -n $(K8S_NS) -f -

migrate-build:
	$(DOCKER) build -t $(MIGRATE_IMAGE) -f $(MIGRATE_DOCKERFILE) .

dev-migrate:
	kubectl --context=$(K8S_CONTEXT) delete job/mcm-migrate -n $(K8S_NS) --ignore-not-found
	$(KUSTOMIZE) $(MIGRATE_OVERLAY_DEV) | kubectl --context=$(K8S_CONTEXT) apply -n $(K8S_NS) -f -
	kubectl --context=$(K8S_CONTEXT) wait --for=condition=complete --timeout=120s -n $(K8S_NS) job/mcm-migrate

prod-migrate:
	kubectl --context=$(K8S_CONTEXT) delete job/mcm-migrate -n $(K8S_NS) --ignore-not-found
	$(KUSTOMIZE) $(MIGRATE_OVERLAY_PROD) | kubectl --context=$(K8S_CONTEXT) apply -n $(K8S_NS) -f -
	kubectl --context=$(K8S_CONTEXT) wait --for=condition=complete --timeout=300s -n $(K8S_NS) job/mcm-migrate
