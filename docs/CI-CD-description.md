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

### Build and publish images manually

1. Export the required environmental variables

```
export IMG="IMG"           // Keda Manager's image
export REGISTRY="REGISTRY" // the OCI registry the module will be published to
```

2. Run the following recipe to build and publish module

```
make module-build \
  IMG=${IMG} \
  REGISTRY={REGISTRY}
```