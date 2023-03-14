PROJECT_COMMON ?= ../common

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: run-with-lifecycle-manager
run-with-lifecycle-manager: ## Create k3d cluster with module installed by the lifecycle-manager.
run-with-lifecycle-manager: run-k3d-with-keda-and-lifecycle-manager busola-enable

.PHONY: run-without-lifecycle-manager
run-without-lifecycle-manager: ## Create k3d cluster with module.
run-without-lifecycle-manager: run-k3d-with-keda-without-lifecycle-manager busola-enable

.PHONY: stop
stop: ## Delete k3d cluster and registry.
	@make -C ${PROJECT_COMMON} delete-k3d

### Internal Dependencies

.PHONY: run-k3d-with-keda-and-lifecycle-manager
run-k3d-with-keda-and-lifecycle-manager:
	@make -C ${PROJECT_COMMON} run-with-lifecycle-manager

.PHONY: run-k3d-with-keda-without-lifecycle-manager
run-k3d-with-keda-without-lifecycle-manager:
	@make -C ${PROJECT_COMMON} run-without-lifecycle-manager

.PHONY: busola-enable
busola-enable:
	@make -C ./busola busola-enable

##@ DEPRECATED

#TODO: remove after update doc and test-infra
.PHONY: run
run: run-with-lifecycle-manager ## Use "run-with-lifecycle-manager" instead.

