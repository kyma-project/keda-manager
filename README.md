# Keda Manager

## Overview 

Keda Manager is an extension to the Kyma ecosystem that allows users to install KEDA. It follows the Kubernetes operator pattern to manage the lifecycle of the KEDA installation based on the existence and the content of the dedicated Keda custom resource (CR).

![a](./docs/assets/keda-overview.drawio.svg)

For more information, see [Use Keda Manager to manage KEDA](/docs/keda-management.md).

### What is KEDA?

KEDA is a flexible Event Driven Autoscaler for the Kubernetes workloads. It extends the Kubernetes autoscaling mechanisms with its own metric server and the possibility to make use of external event sources for making scaling decisions. To learn more about KEDA, see the [KEDA documentation](https://keda.sh/docs/latest/concepts/).

## Install Keda module

1. To install Keda Manager manually, apply the following script:

```bash
kubectl create ns kyma-system
kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
```

1. To get KEDA installed, apply the sample Keda CR:

```bash
kubectl apply -f config/samples/operator_v1alpha1_keda_k3d.yaml
```

For more installation options, check the [Install Keda Manager](/docs/keda-installation.md) tutorial.

##  Development

###  Project structure

Keda Manager codebase is scaffolded with `kubebuilder`. For more information on `kubebuilder`, visit the [project site](https://github.com/kubernetes-sigs/kubebuilder).

- `config`: A directory containing the [kustomize](https://github.com/kubernetes-sigs/kustomize) YAML definitions of the module. For more information, see [kubebuilder's documentation on launch configuration](https://book.kubebuilder.io/cronjob-tutorial/basic-project.html#launch-configuration).
- `api`: Packages containing Keda CustomResourceDefinitions (CRD). 
- `controllers`: Package containing the implementation of the module's reconciliation loop responsible for managing Keda CRs.
- `Dockerfile`: The definition of the `keda-manager-module` image.
- `bin`: A directory with binaries that are used to build/run project.
- `config.yaml`: Configuration file to override module's Helm chart properties.
- `docs`: Contains context documentation for the project.
- `hack`: A directory containing scripts and makefiles that enhance the root `Makefile` capabilities.
- `pkg`: Contains packages used in the project.
- `keda.yaml`: Kubernetes objects that represent `keda module`.


### Prerequisites

- Access to a Kubernetes cluster
- [Go](https://go.dev/)
- [k3d](https://k3d.io/v5.4.6/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)

### Useful Make targets 

You can build and run the Keda Manager in the Kubernetes cluster without Kyma.
For the day-to-day development on your machine, you don't always need to have it controlled by Kyma's Lifecycle Manager.

Run the following commands to deploy Keda Manager on a target Kubernetes cluster (for example, on k3d):

1. Clone the project.

   ```bash
   git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/
   ```

2. Set the Keda Manager image name.

   > NOTE: You can use local k3d registry or your dockerhub account to push intermediate images.  
   ```bash
   export IMG=<DOCKER_USERNAME>/custom-keda-manager:0.0.2
   ```

3. Verify the compatibility.

   ```bash
   make test
   ```
4. Build and push the image to the registry.

   ```bash
   make module-image
   ```
5. Create a target Namespace.

   ```bash
   kubectl create ns kyma-system
   ```

6. Deploy.

   ```bash
   make deploy
   ```

7. Verify if Keda Manager is deployed.

   ```bash
   kubectl get deployments -n kyma-system       
   NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
   keda-manager            1/1     1            1           1m
   ```

## CI/CD

### Pipelines running on pull requests

The following CI jobs are part of the development cycle. They verify the functional correctness of Keda Manager but do not verify the contract concerning Kyma's Lifecycle Manager.

| Name | Required | Description |
|------|----------|-------------|
|[`pre-keda-manager-operator-build`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L144)|true|Builds Keda operator's image and pushes it to the `dev` registry.|
|[`pull-keda-module-build`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L102)|true|Builds module's OCI image and pushes it to the `dev` artifact registry. Renders ModuleTemplate for the Keda module that allows for manual integration tests against Lifecycle Manager.|
|[`pre-keda-manager-operator-tests`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L127)|true|Executes basic create/update/delete functional tests of Keda Manager's reconciliation logic.|
|[`pre-main-keda-manager-verify`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L175)|true|Installs Keda Manager, not using Lifecycle Manager, and applies the sample Keda CR on a k3d cluster. Executes smoke integration test of KEDA.  |
|[`pre-keda-manager-operator-lint`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L61)|false|Is responsible for linting, static code analysis.|

### Pipelines running on main branch 

The following CI jobs are regenerating Keda Manager’s artifacts and initiating integration tests of Keda Manager to verify the contract with respect to Kyma’s Lifecycle Manager.

| Name | Description |
|------|-------------|
|[`post-keda-manager-operator-build`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L158)|Re-builds manager's image and pushes it into the `prod` registry.|
|[`post-keda-module-build`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L80)|Re-builds module's OCI image and pushes it to the `prod` artifact registry.|
|[`post-main-keda-manager-verify`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L193)|Installs Keda Manager, using Lifecyle Manager, applies Kyma CR and enables Keda module on a k3d cluster. Executes smoke integration test of KEDA.|
|[`post-main-keda-manager-upgrade-latest-to-main`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L239)|Installs Keda module, using ModuleTemplate and Lifecycle Manager, from the latest released version and upgrades it to the version from `main`. Verifies reconciliation status on Kyma CR and runs smoke integration tests of KEDA.|

### Building and publishing images manually

- Export the required environmental variables

```
export IMG="IMG"           // Keda Manager's image
export REGISTRY="REGISTRY" // the OCI registry the module will be published to
```

- Run recipe to build and publish module

```
make module-build \
  IMG=${IMG} \
  REGISTRY={REGISTRY}
```

## User inteface

Keda Manager is not only an API extension to the Kyma ecosystem, but it also extends the UI of the Kyma Dashboard.
It uses the [UI extensibility](https://github.com/kyma-project/busola/tree/main/docs/extensibility) feature of Kyma dashboard.
In the [ui-extensions](config/ui-extensions) folder you will find configuration for the UI components (for example, list view, form view, details view) that will help Kyma users manipulate with Keda CRs - `ScaledObjects`.
This configuration is applied as part of the Keda Manager resources. Thanks to that, it comes and goes depending on whether the Keda module is enabled or disabled.

## Releasing new versions 

The release of a new version of the Keda module is realized using the release channels.
This means that new versions are submitted to a given channel.

Current versions per each channel are represented by the [ModuleTemplate CR](https://github.com/kyma-project/lifecycle-manager/blob/main/docs/technical-reference/api/moduleTemplate-cr.md) submitted to a matching folder in the Kyma Git repository.

Having merged all the changes into the `main` branch in the `keda-manager` repository, the CI/CD jobs will bundle module images and generate ModuleTemplate for you.
Submit your ModuleTemplate into the desired channel using a pull request to the Kyma repository.
A series of governance jobs will start testing if the new candidate version fulfills the criteria described in the [module submission process](https://github.com/kyma-project/community/tree/main/concepts/modularization#module-submission-process).


## Keda module footprint

This section describes the impact the installed Keda module has on the cluster resources.

TBD
## Troubleshooting

- For MackBook M1 users

Some parts of the scripts may not work because Kyma CLI is not released for Apple Silicon users. To fix it install [Kyma CLI manually](https://github.com/kyma-project/cli#installation) and export the path to it.

   ```bash
   export KYMA=$(which kyma)
   ```

The example error may look like this: `Error: unsupported platform OS_TYPE: Darwin, OS_ARCH: arm64; to mitigate this problem set variable KYMA with the absolute path to kyma-cli binary compatible with your operating system and architecture. Stop.`
