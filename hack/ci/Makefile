PROJECT_ROOT ?= ../..
PROJECT_COMMON ?= ../common

include ${PROJECT_ROOT}/.env

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
	@cat ${PROJECT_ROOT}/template.yaml
	@echo "\n~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~"

.PHONY: integration-test
integration-test: ## Run integration tests
	cd ${PROJECT_ROOT}/test && go run main.go

.PHONY: k3d-lm-integration-test
k3d-lm-integration-test: ## Run integration tests with lifecycle-manager.
k3d-lm-integration-test: run-with-lifecycle-manager integration-test

.PHONY: k3d-integration-test
k3d-integration-test: ## Run integration tests.
k3d-integration-test: run-without-lifecycle-manager integration-test

.PHONY: k3d-k8s-compatibility-test
k3d-k8s-compatibility-test: ## K8s compatibility tests not implemented yet.
	@echo "k8s compatibility tests not implemented yet"

.PHONY: hyperscalers-compatibility-test
hyperscalers-compatibility-test: ## Hyperscalers compatibility tests not implemented yet.
	@echo "hyperscalers compatibility tests not implemented yet"

.PHONY: upgrade-to
upgrade-to: ## Install Keda onto existing k3d cluster.
	@make -C ${PROJECT_COMMON} reinstall-keda

### Internal Dependencies

.PHONY: run-with-lifecycle-manager
run-with-lifecycle-manager:
	@make -C ${PROJECT_COMMON} run-with-lifecycle-manager

.PHONY: run-without-lifecycle-manager
run-without-lifecycle-manager:
	@make -C ${PROJECT_COMMON} run-without-lifecycle-manager
