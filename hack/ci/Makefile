PROJECT_ROOT ?= ../..
PROJECT_COMMON ?= ../common

MODULE_VERSION ?= 0.0.0

ifndef MODULE_SHA
    MODULE_SHA = ${shell git rev-parse --short HEAD}
endif

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: module-build
module-build: ## Build the Module, push it to a registry and print it based on the commit hash.
	@make -C ${PROJECT_ROOT} module-build \
		IMG=${IMG} \
		MODULE_REGISTRY=${MODULE_REGISTRY} \
		MODULE_VERSION=${MODULE_VERSION}-${MODULE_SHA} \

	@echo "\n~~~~~~~~~~~~BEGINING OF MODULE TEMPLATE~~~~~~~~~~~~~~"
	@cat ${PROJECT_ROOT}/moduletemplate.yaml
	@echo "\n~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~"

.PHONY: integration-test
integration-test: ## Run integration tests
	cd ${PROJECT_ROOT}/test && go run main.go

.PHONY: k3d-lm-integration-test
k3d-lm-integration-test: ## Run integration tests with lifecycle-manager.
k3d-lm-integration-test: run-with-lifecycle-manager integration-test

.PHONY: k3d-integration-test
k3d-integration-test: ## Run integration tests.
k3d-integration-test: run-without-lifecycle-manager integration-test remove-keda

.PHONY: k3d-k8s-compatibility-test
k3d-k8s-compatibility-test: ## K8s compatibility tests not implemented yet.
	@echo "k8s compatibility tests not implemented yet"

.PHONY: hyperscalers-compatibility-test
hyperscalers-compatibility-test: ## Hyperscalers compatibility tests not implemented yet.
	@echo "hyperscalers compatibility tests not implemented yet"

.PHONY: k3d-upgrade-test
k3d-upgrade-test: ## Installs Keda from latest and upgrades to local.
k3d-upgrade-test: \
	install-latest \
	reinstall-keda \
	integration-test

.PHONY: remove-keda
remove-keda: ## Remove Keda CR
	kubectl delete keda -n kyma-system default --timeout 1m || (kubectl get keda -n kyma-system -oyaml && false)

.PHONY: render-manifest
render-manifest:
	@make -C ${PROJECT_ROOT} render-manifest

.PHONY: run-with-lifecycle-manager
run-with-lifecycle-manager:
	@make -C ${PROJECT_COMMON} run-with-lifecycle-manager

.PHONY: run-without-lifecycle-manager
run-without-lifecycle-manager:
	@make -C ${PROJECT_COMMON} run-without-lifecycle-manager

.PHONY: install-latest
install-latest:
	@make -C ${PROJECT_COMMON} kyma create-k3d
	kubectl create ns kyma-system
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-default-cr.yaml -n kyma-system
	@make -C ${PROJECT_COMMON} verify-keda


.PHONY: reinstall-keda
reinstall-keda:
	@make -C ${PROJECT_COMMON} run-without-lm-on-cluster

	# wait some time to make sure lm starts the reconciliation first
	sleep 5

	# double check that serverless controller has progressed
	@make -C ${PROJECT_COMMON} verify-keda