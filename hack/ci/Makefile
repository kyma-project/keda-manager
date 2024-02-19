# local variables

PROJECT_ROOT = ../..
PROJECT_COMMON = ../common

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: integration-test
integration-test: ## Run integration tests
	cd ${PROJECT_ROOT}/test && go run main.go

.PHONY: integration-test-on-cluster
integration-test-on-cluster: ## Run integration tests.
integration-test-on-cluster: run-on-cluster integration-test remove-keda

.PHONY: upgrade-test
upgrade-test: ## Installs Keda from latest and upgrades to local.
upgrade-test: \
	install-latest \
	reinstall-keda \
	integration-test

.PHONY: remove-keda
remove-keda: ## Remove Keda CR
	kubectl delete keda -n kyma-system default --timeout 1m || (kubectl get keda -n kyma-system -oyaml && false)

.PHONY: run-on-cluster
run-on-cluster:
	@make -C ${PROJECT_COMMON} run-on-cluster

.PHONY: install-latest
install-latest:
	kubectl create ns kyma-system || true
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-default-cr.yaml -n kyma-system
	@make -C ${PROJECT_COMMON} verify-keda


.PHONY: reinstall-keda
reinstall-keda:
	@make -C ${PROJECT_COMMON} run-on-cluster

	# wait some time to make sure lm starts the reconciliation first
	sleep 5

	# double check that serverless controller has progressed
	@make -C ${PROJECT_COMMON} verify-keda
