# local variables

PROJECT_ROOT = ../..

include ${PROJECT_ROOT}/hack/k3d.mk

ADDON_NAMESPACE ?= kyma-system
ADDON_VERSION ?= 0.13.0
ADDON_ISTIO_INJECTION ?= false

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
	create-kyma-system-ns \
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

.PHONY: create-kyma-system-ns
create-kyma-system-ns: ## Create kyma-system namespace.
	kubectl create ns kyma-system || true

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

##@ HTTP Add-on

.PHONY: wait-for-keda-ready
wait-for-keda-ready: ## Wait for Keda CR to reach Ready state
	@echo "Waiting for Keda CR to be Ready..."
	@for i in $$(seq 1 30); do \
		STATE=$$(kubectl get keda default -n kyma-system -o jsonpath='{.status.state}' 2>/dev/null || echo ""); \
		if [ "$$STATE" = "Ready" ]; then \
			echo "Keda is Ready"; \
			exit 0; \
		fi; \
		echo "  attempt $$i: state=$$STATE"; \
		sleep 10; \
	done; \
	echo "ERROR: Keda did not become Ready within timeout"; \
	kubectl get keda default -n kyma-system -o yaml; \
	exit 1

.PHONY: enable-addon
enable-addon: ## Enable HTTP add-on via annotations on Keda CR
	kubectl annotate keda default -n kyma-system \
		keda.kyma-project.io/addon-enabled=true \
		keda.kyma-project.io/addon-version=$(ADDON_VERSION) \
		keda.kyma-project.io/addon-namespace=$(ADDON_NAMESPACE) \
		keda.kyma-project.io/addon-istio-injection=$(ADDON_ISTIO_INJECTION) \
		--overwrite

.PHONY: wait-for-addon-ready
wait-for-addon-ready: ## Wait for AddonInstalled condition to become True on Keda CR
	@echo "Waiting for AddonInstalled condition to be True..."
	@for i in $$(seq 1 30); do \
		STATUS=$$(kubectl get keda default -n kyma-system -o jsonpath='{.status.conditions[?(@.type=="AddonInstalled")].status}' 2>/dev/null || echo ""); \
		if [ "$$STATUS" = "True" ]; then \
			echo "AddonInstalled condition is True"; \
			exit 0; \
		fi; \
		REASON=$$(kubectl get keda default -n kyma-system -o jsonpath='{.status.conditions[?(@.type=="AddonInstalled")].reason}' 2>/dev/null || echo ""); \
		echo "  attempt $$i: AddonInstalled status=$$STATUS reason=$$REASON"; \
		sleep 10; \
	done; \
	echo "ERROR: AddonInstalled condition did not become True within timeout"; \
	kubectl get keda default -n kyma-system -o yaml; \
	exit 1


.PHONY: verify-addon-condition
verify-addon-condition: ## Verify that AddonInstalled condition is True on Keda CR
	@ADDON_STATUS=$$(kubectl get keda default -n kyma-system -o jsonpath='{.status.conditions[?(@.type=="AddonInstalled")].status}'); \
	if [ "$$ADDON_STATUS" != "True" ]; then \
		echo "ERROR: AddonInstalled condition status is not True, got: $$ADDON_STATUS"; \
		kubectl get keda default -n kyma-system -o yaml; \
		exit 1; \
	fi; \
	echo "AddonInstalled condition verified: status=True"

.PHONY: disable-addon
disable-addon: ## Disable HTTP add-on via annotation
	kubectl annotate keda default -n kyma-system \
		keda.kyma-project.io/addon-enabled=false \
		--overwrite

.PHONY: wait-for-addon-removal
wait-for-addon-removal: ## Wait for HTTP addon deployments to be removed
	@echo "Waiting for HTTP addon removal from namespace $(ADDON_NAMESPACE)..."
	@for i in $$(seq 1 20); do \
		DEPLOYS=$$(kubectl get deploy -n $(ADDON_NAMESPACE) -l app.kubernetes.io/name=http-add-on --no-headers 2>/dev/null | wc -l); \
		if [ "$$DEPLOYS" -eq 0 ]; then \
			echo "All HTTP addon deployments removed"; \
			exit 0; \
		fi; \
		echo "  attempt $$i: $$DEPLOYS HTTP addon deployments still present"; \
		sleep 10; \
	done; \
	echo "ERROR: HTTP addon deployments were not removed within timeout"; \
	kubectl get deploy -n $(ADDON_NAMESPACE) -l app.kubernetes.io/name=http-add-on; \
	exit 1

.PHONY: verify-addon-disabled
verify-addon-disabled: ## Verify that AddonInstalled condition reason is HTTPAddonDisabled
	@ADDON_REASON=$$(kubectl get keda default -n kyma-system -o jsonpath='{.status.conditions[?(@.type=="AddonInstalled")].reason}'); \
	if [ "$$ADDON_REASON" != "HTTPAddonDisabled" ]; then \
		echo "ERROR: AddonInstalled condition reason is not HTTPAddonDisabled, got: $$ADDON_REASON"; \
		kubectl get keda default -n kyma-system -o yaml; \
		exit 1; \
	fi; \
	echo "AddonInstalled condition verified: reason=HTTPAddonDisabled"
