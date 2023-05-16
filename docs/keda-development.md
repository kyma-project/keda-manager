##  Development

###  Project structure

Keda Manager codebase is scaffolded with `kubebuilder`. For more information on `kubebuilder`, visit the [project site](https://github.com/kubernetes-sigs/kubebuilder).

- `config`: A directory containing the [kustomize](https://github.com/kubernetes-sigs/kustomize) YAML definitions of the module. For more information, see [kubebuilder's documentation on launch configuration](https://book.kubebuilder.io/cronjob-tutorial/basic-project.html#launch-configuration).
- `api`: Packages containing Keda CustomResourceDefinitions (CRD). 
- `controllers`: Package containing the implementation of the module's reconciliation loop responsible for managing Keda custom resources (CRs).
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

   > NOTE: You can use local k3d registry or your Docker Hub account to push intermediate images.  
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

6. Deploy Keda Manager.

   ```bash
   make deploy
   ```

7. Verify if Keda Manager is deployed.

   ```bash
   kubectl get deployments -n kyma-system       
   NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
   keda-manager            1/1     1            1           1m
   ```