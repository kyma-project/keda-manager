CLUSTER_NAME ?= kyma
REGISTRY_PORT ?= 5001
REGISTRY_NAME ?= ${CLUSTER_NAME}-registry

MANAGER_IMAGE_NAME ?= keda-manager-dev-local
MANAGER_IMAGE_TAG ?= 0.0.1

PROJECT_ROOT ?= ../../..

TEMPLATE_PATH ?= ${PROJECT_ROOT}/template.yaml

KYMA ?= ${PROJECT_ROOT}/bin/kyma-unstable

.PHONY: kyma
kyma:
	@make -C ${PROJECT_ROOT} kyma

.PHONY: delete-k3d-registry
delete-k3d-registry:
	-k3d registry delete ${REGISTRY_NAME}

.PHONY: delete-k3d-cluster
delete-k3d-cluster:
	-k3d cluster delete ${CLUSTER_NAME}

.PHONY: create-k3d
create-k3d: 
	${KYMA} provision k3d --registry-port ${REGISTRY_PORT} --name ${CLUSTER_NAME} --ci

.PHONY: delete-k3d
delete-k3d: delete-k3d-cluster delete-k3d-registry

.PHONY: create-kyma-system-ns
create-kyma-system-ns:
	kubectl create ns kyma-system

.PHONY: module-image
module-image:
	@make -C ${PROJECT_ROOT} module-image \
		IMG=localhost:${REGISTRY_PORT}/${MANAGER_IMAGE_NAME}:${MANAGER_IMAGE_TAG}

.PHONY: module-build
module-build:
	@make -C ${PROJECT_ROOT} module-build \
		IMG=k3d-${REGISTRY_NAME}:${REGISTRY_PORT}/${MANAGER_IMAGE_NAME}:${MANAGER_IMAGE_TAG} \
		MODULE_REGISTRY=localhost:${REGISTRY_PORT}/unsigned

.PHONY: yq-install
yq-install:
	@make -C ${PROJECT_ROOT} yq

.PHONY: fix-template
fix-template:
	@cat ${PROJECT_ROOT}/template.yaml \
	| sed -e 's/remote/control-plane/g' \
		-e 's/${REGISTRY_PORT}/5000/g' \
	      	-e 's/localhost/k3d-${REGISTRY_NAME}.localhost/g' \
	> ${PROJECT_ROOT}/template-k3d.yaml

.PHONY: enable-module
enable-module:
	kubectl patch kymas.operator.kyma-project.io \
		default-kyma \
		-n kcp-system \
		--type=merge \
		--patch-file ./k3d-patches/patch-kyma.yaml

.PHONY: patch-mod-mgr-role
patch-mod-mgr-role:
	kubectl patch clusterrole \
		module-manager-manager-role \
		--patch-file=./k3d-patches/patch-k3d-module-manager-clusterrole.yaml

.PHONY: install-module
install-module:
	${KYMA} alpha deploy \
		--ci \
		--template=${PROJECT_ROOT}/template-k3d.yaml

.PHONY: run
run: kyma \
	create-k3d \
	create-kyma-system-ns \
	module-image \
	yq-install \
	module-build \
	fix-template \
	install-module \
	patch-mod-mgr-role \
	enable-module verify

.PHONY: stop
stop: delete-k3d

.PHONY: verify
verify:
	@${PROJECT_ROOT}/hack/verify_kyma_status.sh

