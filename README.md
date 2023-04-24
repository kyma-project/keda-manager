# Keda Manager

## Overview 

Keda Manager is an extension to the Kyma ecosystem that allows users to install KEDA. It follows the Kubernetes operator pattern to manage the lifecycle of the KEDA installation based on the existence and the content of the dedicated Keda custom resource (CR).

![a](./docs/assets/keda-overview.drawio.svg)

### What is KEDA?

KEDA is a flexible Event Driven Autoscaler for the Kubernetes workloads. It extends the Kubernetes autoscaling mechanisms with its own metric server and the possibility to make use of external event sources to make scaling decisions. To learn more about KEDA, see [KEDA documentation](https://keda.sh/docs/latest/concepts/).

## Install

For the installation options check the [How to install Keda Manager](/docs/keda-installation.md) tutorial.

##  Development

###  Project structure

Keda Manager codebase is scaffolded with `kubebuilder`. For more information on `kubebuilder`, visit the [project site](https://github.com/kubernetes-sigs/kubebuilder)).

- `config`: A directory containing the [kustomize](https://github.com/kubernetes-sigs/kustomize) YAML definitions of the module for more information, see [kubebuilder's documentation on launch configuration](https://book.kubebuilder.io/cronjob-tutorial/basic-project.html#launch-configuration)).
- `api`: Packages containing Keda CustomResourceDefinitions (CRD). 
- `controllers`: Package containing the implementation of the module's reconciliation loop responsible for managing Keda CRs.
- `Dockerfile`: The definition of the `keda-manager-module` image.
- `bin`: A directory with binaries that are used to build/run project.
- `config.yaml`: Configuration file to override module's Helm chart properties.
- `docs`: Contains context documentation for the project.
- `hack`: A directory containing scripts and makefiles that enchance capabilities of root `Makefile`.
- `pkg`: Contains packages used in the project.
- `keda.yaml`: Kubernetes objects that represent `keda module`


### Prerequisites

- Access to a Kubernetes cluster
- [Go](https://go.dev/)
- [k3d](https://k3d.io/v5.4.6/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)

### Useful Make targets 

You can build and run the Keda Manager in the Kubernetes cluster without Kyma.
For the day-to-day development on your machine, you don't always need to have it controlled by Kyma's `lifecycle-manager`.

Run the following commands to deploy Keda Manager on a target Kubernetes cluster (i.e., on k3d):

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

7. Verify if Keda Manager is deployed

   ```bash
   kubectl get deployments -n kyma-system       
   NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
   keda-manager            1/1     1            1           1m
   ```


### How to use Keda Manager to install Keda

Keda Manager reconciles KEDA deployment based on the watched Keda CRs:

- Apply Keda CR (sample) to have Keda installed.

   ```bash
   kubectl apply -f config/samples/operator_v1alpha1_keda_k3d.yaml
   ```

   After a while, you will have Keda installed, and you should see its workloads:

   ```bash
   NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
   keda-manager                     1/1     1            1           3m
   keda-operator                    1/1     1            1           3m
   keda-operator-metrics-apiserver  1/1     1            1           3m
   ```

   Now you can use KEDA to scale workloads on the Kubernetes cluster. Check the [demo application](docs/keda-demo-application.md).

- Remove Keda CR to have Keda uninstalled.

   ```bash
   kubectl delete -f config/samples/operator_v1alpha1_keda_k3d.yaml
   ```
   This uninstalls all Keda workloads but leaves `keda-manager`.

   > **NOTE:** Keda Manager uses finalizers to uninstall the Keda module from the cluster. It means that Keda Manager blocks the uninstallation process of KEDA until there are user-created custom resources (for example ScaledObjects).

- Update the specification of Keda CR to change the Keda installation

   [This example](docs/keda-configuration.md) shows how to modify the Keda properties using the `keda.operator.kyma-project.io` CR.


   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: operator.kyma-project.io/v1alpha1
   kind: Keda
   metadata:
   name: default
   spec:
   logging:
      operator:
         level: "debug"
   resources:
      operator:
         limits:
         cpu: "1"
         memory: "200Mi"
         requests:
         cpu: "0.5"
         memory: "150Mi"
      metricServer:
         limits:
         cpu: "1"
         memory: "1000Mi"
         requests:
         cpu: "300m"
         memory: "500Mi"
   EOF
   ```

## CI/CD

### Pipelines running on pull requests

The following CI jobs are part of the development cycle. They verify the functional correctness of keda-manager but do not verify the contract concerning Kyma's lifecycle-manager.

| Name | Required | Description |
|------|----------|-------------|
|[`pre-keda-manager-operator-build`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L144)|true|builds Keda operator's image and pushes it to dev registry|
|[`pull-keda-module-build`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L102)|true|builds module's OCI image and pushes it to dev artifact registry. Renders Module Template for the Keda module that allows for manual integration tests against lifecycle-manager|
|[`pre-keda-manager-operator-tests`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L127)|true|executes basic create/update/delete functional tests of the keda-manager's reconciliation logic|
|[`pre-main-keda-manager-verify`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L175)|true|installs keda-manager (**not using  lifecycle-manager**) and applies sample Keda CR on k3d cluster. Executes smoke integration test of Keda.  |
|[`pre-keda-manager-operator-lint`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L61)|false|linting, static code analysis|

### Pipelines running on main branch 

The following CI jobs are regenerating keda-manager's artefacts and initiate integration tests of keda-manager to verify contract with respect to Kyma's lifecycle-manager.

| Name | Description |
|------|-------------|
|[`post-keda-manager-operator-build`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L158)|re-builds manager's image and pushes it into prod registry|
|[`post-keda-module-build`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L80)|re-builds module's OCI image and pushes it to prod artifact registry|
|[`post-main-keda-manager-verify`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L193)|installs keda-manager (**using lifecycle-manager**), applies Kyma CR and enables keda module on k3d cluster. Executes smoke integration test of Keda.|
|[`post-main-keda-manager-upgrade-latest-to-main`](https://github.com/kyma-project/test-infra/blob/main/templates/data/generic_module_data.yaml#L239)|installs keda module (using module template and lifecycle-manager) from latest released version and upgrades it to the version from main. Verifies reconciliation status on the Kyma CR and runs smoke integration tests of keda|

### Building and publishing images manually

- Export required environmental variables

```
export IMG="IMG"           // keda manager's image
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
In the [ui-extensions](config/ui-extensions) folder you will find configuration for the UI components (i.e., list view, form view, details view) that will help Kyma users manipulate with Keda CRs - `ScaledObjects`.
This configuration is applied as part of the Keda Manager resources. Thanks to that, it comes and goes depending on whether the Keda module is enabled or disabled.

## Releasing new versions 

The release of a new version of the Keda module is realized using the [release channels](https://github.com/kyma-project/community/tree/main/concepts/modularization#release-channels).
This means that new versions are submitted to a given channel.

Current versions per each channel are represented by the Module Templates CR submitted to a matching folder in the Kyma git repository:

 - fast (not available yet)
 - regular (not available yet)

Having merged all the changes into the main branch in the `keda-manager` repository, the CI/CD jobs will bundle module images and generate a module template for you.
Take the module template and submit it into the desired channel using a pull request to the Kyma repository.
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
