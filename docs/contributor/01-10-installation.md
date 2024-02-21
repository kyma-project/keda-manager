# Install Keda Manager 

- [Install Keda Manager](#install-keda-manager)
  - [Install Keda Manager from the local sources](#install-keda-manager-from-the-local-sources)
    - [Prerequisites](#prerequisites)
    - [Procedure](#procedure)
  - [Make targets to run Keda Manager locally on k3d](#make-targets-to-run-keda-manager-locally-on-k3d)
    - [Run Keda Manager](#run-keda-manager)

Learn how to install the Keda module locally (on k3d) or on your remote cluster.

## Install Keda Manager From the Local Sources 

### Prerequisites

- Access to a Kubernetes (v1.24 or higher) cluster
- [Go](https://go.dev/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)

### Procedure

You can build and run the Keda Manager in the Kubernetes cluster without Kyma.
For the day-to-day development on your machine, you don't always need to have it controlled by Kyma's Lifecycle Manager.

Run the following commands to deploy Keda Manager in a target Kubernetes cluster, such as k3d:

1. Clone the project.

   ```bash
   git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/
   ```

2. Set the Keda Manager image name.

   > NOTE: You can use the local k3d registry or your Docker Hub account to push intermediate images.  
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
5. Create a target namespace.

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
   ```

   You should get a result similar to this example:

   ```
   NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
   keda-manager            1/1     1            1           1m
   ```

## Make Targets To Run Keda Manager Locally on k3d

### Run Keda Manager

When using a local k3d cluster, you can also use the local OCI image registry that comes with it.
Thanks to that, you don't need to push the Keda module images to a remote registry and you can test the changes in the Kyma installation set up entirely on your machine.

1. Clone the project.

   ```bash
   git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/
   ```
2. Build the manager locally and run it in the k3d cluster.

   ```bash
   make -C hack/local run
   ```
3. If you want to clean up the k3d cluster, use the `make -C hack/local stop` make target.

