PROJECT_ROOT ?= ../..
PROJECT_KEDA_LOCAL ?= ${PROJECT_ROOT}/hack/local/keda

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
	@echo "\n~~~~~~~~~~~~BEGINING OF MODULE TEMPLATE~~~~~~~~~~~~~~"
	@cat ${PROJECT_ROOT}/template.yaml
	@echo "\n~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~"

# rename it from 'all' to something more accurate like 'module-build-and-print'
.PHONY: all
all: module-build module-template-print

.PHONY: integration-test
integration-test:
	echo "tests not implemented yet"

.PHONY: k3d-integration-test
k3d-integration-test:
	make -C ${PROJECT_KEDA_LOCAL} run
	make integration-test
