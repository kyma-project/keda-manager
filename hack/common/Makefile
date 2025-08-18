# local variables

PROJECT_ROOT = ../..

include ${PROJECT_ROOT}/hack/k3d.mk

##@ General

.PHONY: help
## Display this help.
help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: run
run: ## Create k3d cluster and deploy module.
run: \
	create-k3d \
	module-image \
	deploy \
	apply-keda \
	verify-keda

.PHONY: run-on-cluster
run-on-cluster: ## Deploy module on existing cluster.
run-on-cluster: \
	deploy-cluster \
	apply-keda \
	verify-keda


##@ Module management

.PHONY: verify-keda
verify-keda: ## Wait for Keda CR to be in Ready state.
	sleep 10

	./verify_keda_status.sh || \
		(make print-keda-details && false)

	kubectl wait --for condition=Available deployment keda-operator -n kyma-system --timeout=60s || \
		(make print-keda-details && false)

	kubectl wait --for condition=Available deployment keda-manager -n kyma-system --timeout=60s || \
		(make print-keda-details && false)

.PHONY: print-keda-details
print-keda-details: ## Print all pods, deploys and serverless CRs in the kyma-system namespace.
	kubectl get keda -n kyma-system -oyaml
	kubectl get deploy -n kyma-system -oyaml
	kubectl get pods -n kyma-system -oyaml

.PHONY: apply-keda
apply-keda: ## Apply the k3d serverless CR
	kubectl apply -n kyma-system \
		-f ${PROJECT_ROOT}/config/samples/keda-default-cr.yaml

### Internal Dependencies

.PHONY: kyma
kyma:
	@make -C ${PROJECT_ROOT} kyma

.PHONY: module-image
module-image: 
	@make -C ${PROJECT_ROOT} module-image \
		IMG=localhost:${REGISTRY_PORT}/keda-manager-dev-local:0.0.0

.PHONY: deploy
deploy:
	@make -C ${PROJECT_ROOT} deploy \
		IMG=k3d-${REGISTRY_NAME}:${REGISTRY_PORT}/keda-manager-dev-local:0.0.0

.PHONY: deploy-cluster
deploy-cluster:
	@make -C ${PROJECT_ROOT} deploy
