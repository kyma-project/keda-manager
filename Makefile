# incoming variables

KEDA_VERSION ?= 2.18.3

MODULE_VERSION ?= 0.0.0

# Image URL to use all building/pushing image targets
IMG ?= europe-docker.pkg.dev/kyma-project/prod/keda-manager:$(MODULE_VERSION)

# Operating system architecture
OS_ARCH ?= $(shell uname -m)

# Operating system type
OS_TYPE ?= $(shell uname)


# local variables

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.34.1

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: keda-manifest
keda-manifest: ## Upgrade keda.yaml based on the KEDA_VERSION env and chart from the https://github.com/kedacore/charts/tree/main/keda
	helm repo add kedacore https://kedacore.github.io/charts
	helm repo update
	helm template keda kedacore/keda --version $(KEDA_VERSION) --namespace kyma-system --values hack/keda_values.yaml  > keda.yaml

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_CONTROLPLANE_START_TIMEOUT=2m KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT=2m KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	LOG_CONFIG_PATH=./hack/keda-log-config.yaml go run ./main.go

.PHONY: docker-build
docker-build: manifests generate ## Build docker image with the manager.
	IMG=$(IMG) docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

IGNORE_NOT_FOUND ?= false

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with IGNORE_NOT_FOUND=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(IGNORE_NOT_FOUND) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: render-manifest
render-manifest: kustomize ## Render keda-manager.yaml manifest.
	$(KUSTOMIZE) build config/default > keda-manager.yaml

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with IGNORE_NOT_FOUND=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(IGNORE_NOT_FOUND) -f -

##@ Module

.PHONY: module-image
module-image: docker-build docker-push ## Build the Module Image and push it to a registry defined in IMG
	echo "built and pushed module image $(IMG)"


##@ Tools

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

########## Kustomize ###########
KUSTOMIZE ?= $(LOCALBIN)/kustomize
KUSTOMIZE_VERSION ?= v5.5.0
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	@if [ -z ${GITHUB_TOKEN} ]; then \
		test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }; \
	else \
		test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) --header "Authorization: Bearer ${GITHUB_TOKEN}" | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }; \
	fi

########## Controller-Gen ###########
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.19.0

.PHONY: controller-gen $(CONTROLLER_GEN)
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test "$(shell ${LOCALBIN}/controller-gen --version)" = "Version: ${CONTROLLER_TOOLS_VERSION}" || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

########## Envtest ###########
ENVTEST ?= $(LOCALBIN)/setup-envtest

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest


########## Grafana Dashboard ###########
.PHONY: grafana-dashboard
grafana-dashboard: ## Generating Grafana manifests to visualize controller status.
	cd operator && kubebuilder edit --plugins grafana.kubebuilder.io/v1-alpha

##@ Actions
.PHONY: module-config
module-config:
	yq ".version = \"${MODULE_VERSION}\" "\
    module-config-template.yaml > module-config.yaml
