# Testing Strategy

The following CI jobs are part of the development cycle. They verify the functional correctness of Keda Manager but do not verify the contract concerning Kyma's Lifecycle Manager.

## CI/CD Jobs Running on Pull Requests

- `markdown / documentation-link-check` - Checks if there are no broken links in the pull request `.md` files. For the configuration, see the [mlc.config.json](https://github.com/kyma-project/keda-manager/blob/main/mlc.config.json) and the [markdown.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/markdown.yaml) files.
- `lint / lint` - Is responsible for the Keda Operator linting and static code analysis. For the configuration, see the [lint.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/lint.yaml) file.
- `unit tests / unit-tests` - Runs basic unit tests of Keda Operator's logic. For the configuration, see the [unit-tests.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/unit-tests.yaml) file.
- `integration tests / integration-test` - Runs the basic functionality integration test suite for the Keda Operator in a k3d cluster. For the configuration, see the [integration-tests.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/integration-tests.yaml) file.
- `gitleaks / gitleaks-scan` - Scans the pull request for secrets and credentials. For the configuration, see the [gitleaks.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/gitleaks.yaml) file. 

## CI/CD Jobs Running on the Main Branch

- `upgrade tests / upgrade-test`- Runs the upgrade integration test suite and verifies if the latest release can be successfully upgraded to the new (`main`) revision. For the configuration, see the [upgrade-tests.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/upgrade-tests.yaml) file.

## CI/CD Jobs Running on a Schedule

- `markdown / documentation-link-check` - Runs Markdown link check every day at 05:00 AM. For the configuration, see the [markdown.yaml](https://github.com/kyma-project/keda-manager/blob/main/.github/workflows/markdown.yaml) file.
