PROJECT_ROOT ?= ../..

.PHONY: kyma
kyma:
	@make -C ${PROJECT_ROOT} kyma

.PHONY: module-build
module-build: 
	@make -C ${PROJECT_ROOT} module-build \
		IMG=${IMG} \
		MODULE_REGISTRY=${MODULE_REGISTRY} \
		MODULE_VERSION=${MODULE_VERSION}

.PHONY: module-template-print
module-template-print:
	@cat ${PROJECT_ROOT}/template.yaml

.PHONY: all
all: module-build module-template-print
