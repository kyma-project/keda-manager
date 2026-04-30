# local variables

PROJECT_ROOT = ../..
PROJECT_COMMON = ../common

ADDON_NAMESPACE ?= keda
ADDON_VERSION ?= 0.13.0
KEDA_CR_NAMESPACE ?= kyma-system
KEDA_CR_NAME ?= default

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
	@make -C ${PROJECT_COMMON} create-kyma-system-ns
	@make -C ${PROJECT_COMMON} run-on-cluster

.PHONY: install-latest
install-latest:
	kubectl create ns kyma-system || true
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-default-cr.yaml -n kyma-system
	@make -C ${PROJECT_COMMON} verify-keda


.PHONY: reinstall-keda
reinstall-keda: run-on-cluster
	# wait some time to make sure lm starts the reconciliation first
	sleep 5

	# double check that serverless controller has progressed
	@make -C ${PROJECT_COMMON} verify-keda

##@ HTTP Add-on

.PHONY: wait-for-keda-ready
wait-for-keda-ready: ## Wait for Keda CR to reach Ready state
	@echo "Waiting for Keda CR to be Ready..."
	@for i in $$(seq 1 30); do \
		STATE=$$(kubectl get keda $(KEDA_CR_NAME) -n $(KEDA_CR_NAMESPACE) -o jsonpath='{.status.state}' 2>/dev/null || echo ""); \
		if [ "$$STATE" = "Ready" ]; then \
			echo "Keda is Ready"; \
			exit 0; \
		fi; \
		echo "  attempt $$i: state=$$STATE"; \
		sleep 10; \
	done; \
	echo "ERROR: Keda did not become Ready within timeout"; \
	kubectl get keda $(KEDA_CR_NAME) -n $(KEDA_CR_NAMESPACE) -o yaml; \
	exit 1

.PHONY: enable-addon
enable-addon: ## Enable HTTP add-on via annotations on Keda CR
	kubectl annotate keda $(KEDA_CR_NAME) -n $(KEDA_CR_NAMESPACE) \
		keda.kyma-project.io/addon-enabled=true \
		keda.kyma-project.io/addon-version=$(ADDON_VERSION) \
		keda.kyma-project.io/addon-namespace=$(ADDON_NAMESPACE) \
		--overwrite

.PHONY: wait-for-addon-ready
wait-for-addon-ready: ## Wait for addon deployments and pods to be ready
	@echo "Waiting for addon deployments in namespace $(ADDON_NAMESPACE)..."
	@for i in $$(seq 1 30); do \
		DEPLOYS=$$(kubectl get deploy -n $(ADDON_NAMESPACE) --no-headers 2>/dev/null | wc -l); \
		if [ "$$DEPLOYS" -ge 3 ]; then \
			echo "Found $$DEPLOYS deployments"; \
			break; \
		fi; \
		echo "  attempt $$i: $$DEPLOYS deployments (3 expected)"; \
		sleep 10; \
	done
	kubectl wait --for=condition=Available deployment --all -n $(ADDON_NAMESPACE) --timeout=180s
	kubectl wait --for=condition=Ready pod --all -n $(ADDON_NAMESPACE) --timeout=120s

.PHONY: verify-addon-condition
verify-addon-condition: ## Verify that Addon condition is True on Keda CR
	@ADDON_STATUS=$$(kubectl get keda $(KEDA_CR_NAME) -n $(KEDA_CR_NAMESPACE) -o jsonpath='{.status.conditions[?(@.type=="Addon")].status}'); \
	if [ "$$ADDON_STATUS" != "True" ]; then \
		echo "ERROR: Addon condition status is not True, got: $$ADDON_STATUS"; \
		kubectl get keda $(KEDA_CR_NAME) -n $(KEDA_CR_NAMESPACE) -o yaml; \
		exit 1; \
	fi; \
	echo "Addon condition verified: status=True"

.PHONY: disable-addon
disable-addon: ## Disable HTTP add-on via annotation
	kubectl annotate keda $(KEDA_CR_NAME) -n $(KEDA_CR_NAMESPACE) \
		keda.kyma-project.io/addon-enabled=false \
		--overwrite

.PHONY: wait-for-addon-removal
wait-for-addon-removal: ## Wait for addon deployments to be removed
	@echo "Waiting for addon removal from namespace $(ADDON_NAMESPACE)..."
	@for i in $$(seq 1 20); do \
		DEPLOYS=$$(kubectl get deploy -n $(ADDON_NAMESPACE) --no-headers 2>/dev/null | wc -l); \
		if [ "$$DEPLOYS" -eq 0 ]; then \
			echo "All addon deployments removed"; \
			exit 0; \
		fi; \
		echo "  attempt $$i: $$DEPLOYS deployments still present"; \
		sleep 10; \
	done; \
	echo "ERROR: Addon deployments were not removed within timeout"; \
	kubectl get deploy -n $(ADDON_NAMESPACE); \
	exit 1

.PHONY: http-addon-integration-test
http-addon-integration-test: ## Run full HTTP add-on integration test (enable, verify, disable, verify removal)
http-addon-integration-test: wait-for-keda-ready enable-addon wait-for-addon-ready verify-addon-condition disable-addon wait-for-addon-removal
	@echo "HTTP add-on integration test passed"