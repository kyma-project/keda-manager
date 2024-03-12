# Project Structure

Keda Manager codebase is scaffolded with `kubebuilder`. For more information on `kubebuilder`, visit the [project site](https://github.com/kubernetes-sigs/kubebuilder).

- `config`: A directory containing the [kustomize](https://github.com/kubernetes-sigs/kustomize) YAML definitions of the module. For more information, see [kubebuilder's documentation on launch configuration](https://book.kubebuilder.io/cronjob-tutorial/basic-project.html#launch-configuration).
- `api`: Packages containing Keda CustomResourceDefinitions (CRD).
- `controllers`: Package containing the implementation of the module's reconciliation loop responsible for managing Keda custom resources (CRs).
- `Dockerfile`: The definition of the `keda-manager-module` image.
- `bin`: A directory with binaries that are used to build/run project.
- `config.yaml`: Configuration file to override the module's Helm chart properties.
- `docs`: Contains context documentation for the project.
- `hack`: A directory containing scripts and makefiles that enhance the root `Makefile` capabilities.
- `pkg`: Contains packages used in the project.
- `keda.yaml`: Kubernetes objects that represent `keda module`.
