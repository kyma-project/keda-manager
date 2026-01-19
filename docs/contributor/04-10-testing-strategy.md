# Testing Strategy

The following CI jobs are part of the development cycle. They verify the functional correctness of Keda Manager but do not verify the contract concerning Kyma's Lifecycle Manager.

## CI/CD Jobs Running on Pull Requests

- `lint / lint` - Is responsible for the Keda Operator linting and static code analysis. For the configuration, see the [lint.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/lint.yaml) file.
- `pull / unit tests / unit-tests` - Runs basic unit tests of Keda Operator's logic. For the configuration, see the [unit-tests.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/_unit-tests.yaml) file.
- `pull / integration tests / integration-test` - Runs the basic functionality integration test suite for the Keda Operator in a k3d cluster. For the configuration, see the [integration-tests.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/_integration-tests.yaml) file.
- `pull / gitleaks / gitleaks-scan` - Scans the pull request for secrets and credentials. For the configuration, see the [gitleaks.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/_gitleaks.yaml) file. 

## CI/CD Jobs Running on the Main Branch

- `markdown / documentation-link-check` - Checks if there are no broken links in `.md` files. For the configuration, see the [mlc.config.json](https://github.com/kyma-project/keda-manager/blob/main/.mlc.config.json) and the [markdown.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/markdown.yaml) files.
- `push / upgrade tests / upgrade-test`- Runs the upgrade integration test suite and verifies if the latest release can be successfully upgraded to the new (`main`) revision. For the configuration, see the [upgrade-tests.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/_upgrade-tests.yaml) file.


## Smoke-Test the Keda Module on Your Cluster

Follow these steps to verify that the Keda module works on your Kyma instance:
1. Clone the [keda-manager repository](https://github.com/kyma-project/keda-manager) locally.
2. Point the KUBECONFIG environment variable to the file containing the kubeconfig configuration of your cluster.

```
export KUBECONFIG=<path-to-kubeconfig>
```

3. Check if the `Keda` custom resource is in the `Ready` state using the following command:

```
kubectl get kedas.operator.kyma-project.io -n kyma-system
NAME      GENERATION   AGE   STATE
default   1            13d   Ready
```

4. Run the tests using the following make targets in the root of the cloned repository:

```
make -C hack/ci integration-test-on-cluster
```