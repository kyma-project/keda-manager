PROJECT_ROOT ?= ../..

include ${PROJECT_ROOT}/.env

ifndef MODULE_SHA
    MODULE_SHA = ${shell git rev-parse --short HEAD}
endif

.PHONY: kyma
kyma:
	@make -C ${PROJECT_ROOT} kyma

.PHONY: module-build
module-build: 
	@make -C ${PROJECT_ROOT} module-build \
		IMG=${IMG} \
		MODULE_REGISTRY=${MODULE_REGISTRY} \
		MODULE_VERSION=${MODULE_VERSION}-${MODULE_SHA} \

.PHONY: module-template-print
module-template-print:
	@echo "~~~~~~~~~~~~BEGINING OF MODULE TEMPLATE~~~~~~~~~~~~~~"
	@cat ${PROJECT_ROOT}/template.yaml
	@echo "~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~~~~"

.PHONY: all
all: module-build module-template-print
