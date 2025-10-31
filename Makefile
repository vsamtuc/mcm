KUSTOMIZE ?= kubectl kustomize
K8S_NS ?= mcm

K8S_CONTEXT ?= mcm

.PHONY: dev-apply dev-diff prod-apply prod-diff

dev-apply:
	$(KUSTOMIZE) deploy/kustomize/overlays/dev | kubectl --context=$(K8S_CONTEXT) apply -n $(K8S_NS) -f -

dev-diff:
	$(KUSTOMIZE) deploy/kustomize/overlays/dev | kubectl --context=$(K8S_CONTEXT) diff -n $(K8S_NS) -f -

prod-apply:
	$(KUSTOMIZE) deploy/kustomize/overlays/prod | kubectl --context=$(K8S_CONTEXT) apply -n $(K8S_NS) -f -
