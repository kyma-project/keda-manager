PROJECT_ROOT ?= ../..
ifndef MODULE_VERSION
    include ${PROJECT_ROOT}/.version
endif

CLUSTER_NAME ?= kyma
REGISTRY_PORT ?= 5001
REGISTRY_NAME ?= ${CLUSTER_NAME}-registry

MANAGER_IMAGE_NAME ?= keda-manager-dev-local
MANAGER_IMAGE_TAG ?= $(MODULE_VERSION)


KYMA ?= ${PROJECT_ROOT}/bin/kyma-unstable

##@ General

.PHONY: help
## Display this help.
help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: run-with-lifecycle-manager
run-with-lifecycle-manager: ## Create k3d cluster and deploy module with the lifecycle-manager.
run-with-lifecycle-manager: \
    kyma \
	create-k3d \
	create-kyma-system-ns \
	module-image \
	module-build \
	fix-template \
	install-kyma-with-lm \
	patch-mod-mgr-role \
	install-module-template \
	enable-module \
	verify-kyma

.PHONY: run-without-lifecycle-manager
run-without-lifecycle-manager: ## Create k3d cluster and deploy module without the lifecycle-manager.
run-without-lifecycle-manager: \
    kyma \
	create-k3d \
	create-kyma-system-ns \
	module-image \
	deploy \
	apply-keda \
	verify-keda

##@ K3D

.PHONY: create-k3d
create-k3d: ## Create k3d with kyma CRDs.
	${KYMA} provision k3d --registry-port ${REGISTRY_PORT} --name ${CLUSTER_NAME} --ci

.PHONY: delete-k3d
delete-k3d: ## Delete k3d with kyma CRDs.
delete-k3d: delete-k3d-cluster delete-k3d-registry

.PHONY: delete-k3d-registry
delete-k3d-registry: ## Delete k3d kyma registry.
	-k3d registry delete ${REGISTRY_NAME}

.PHONY: delete-k3d-cluster
delete-k3d-cluster: ## Delete k3d kyma cluster.
	-k3d cluster delete ${CLUSTER_NAME}

##@ Module management

.PHONY: verify-keda
verify-keda: ## Wait for Keda CR to be in Ready state.
	@./verify_keda_status.sh

.PHONY: verify-kyma
verify-kyma: ## Wait for Kyma CR to be in Ready state.
	@./verify_kyma_status.sh

.PHONY: fix-template
fix-template: ## Create moduletemplate-k3d.yaml based on moduletemplate.yaml with right URLs.
	@cat ${PROJECT_ROOT}/moduletemplate.yaml \
	| sed -e 's/${REGISTRY_PORT}/5000/g' \
	    -e 's/localhost/k3d-${REGISTRY_NAME}.localhost/g' \
		-e 's/moduletemplate-keda/keda-fast/g' \
	> ${PROJECT_ROOT}/moduletemplate-k3d.yaml

.PHONY: enable-module
enable-module:
	${KYMA} alpha enable module keda -c fast

.PHONY: patch-mod-mgr-role
patch-mod-mgr-role: ## Path lifecycle-manager clusterrole.
	kubectl patch clusterrole \
		lifecycle-manager-manager-role \
		--patch-file=./k3d-patches/patch-k3d-lifecycle-manager-clusterrole.yaml

.PHONY: install-kyma-with-lm
install-kyma-with-lm: ## Install Kyma with lifecycle-manager.
	${KYMA} alpha deploy \
		--ci \
		--force-conflicts
	kubectl label -n kyma-system kyma default-kyma operator.kyma-project.io/beta=true

.PHONY: install-module-template
install-module-template: ## Apply k3d-ready moduletemplate.
	kubectl apply -f ${PROJECT_ROOT}/moduletemplate-k3d.yaml

.PHONY: apply-keda
apply-keda: ## Apply the k3d serverless CR
	kubectl apply -n kyma-system \
		-f ${PROJECT_ROOT}/config/samples/keda-default-cr.yaml

.PHONY: create-kyma-system-ns
create-kyma-system-ns: ## Create kyma-system namespace.
	kubectl create ns kyma-system

### Internal Dependencies

.PHONY: kyma
kyma:
	@make -C ${PROJECT_ROOT} kyma

.PHONY: module-image
module-image: 
	@make -C ${PROJECT_ROOT} module-image \
		IMG=localhost:${REGISTRY_PORT}/${MANAGER_IMAGE_NAME}:${MANAGER_IMAGE_TAG}

.PHONY: clean-charts
clean-charts:
	@rm -rf ${PROJECT_ROOT}/charts

.PHONY: module-build
module-build: clean-charts
	@make -C ${PROJECT_ROOT} module-build \
		PROW_JOB_ID= \
		IMG=k3d-${REGISTRY_NAME}:${REGISTRY_PORT}/${MANAGER_IMAGE_NAME}:${MANAGER_IMAGE_TAG} \
		MODULE_REGISTRY=localhost:${REGISTRY_PORT} \

	@echo "\n~~~~~~~~~~~~BEGINING OF MODULE TEMPLATE~~~~~~~~~~~~~~"
	@cat ${PROJECT_ROOT}/moduletemplate.yaml
	@echo "\n~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~"

.PHONY: deploy
deploy:
	@make -C ${PROJECT_ROOT} deploy \
		IMG=k3d-${REGISTRY_NAME}:${REGISTRY_PORT}/${MANAGER_IMAGE_NAME}:${MANAGER_IMAGE_TAG}

