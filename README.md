# Keda Manager

## Overview

Keda Manager is a module compatible with Lifecycle Manager that allows you to add KEDA Event Driven Autoscaler to the Kyma ecosystem. 

See also:
- [lifecycle-manager documetation](https://github.com/kyma-project/lifecycle-manager)
- [KEDA documentation](https://keda.sh/docs/2.7/concepts/)

## Prerequisites

- Access to a k8s cluster
- [Go](https://go.dev/)
- [k3d](https://k3d.io/v5.4.6/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)

## Installation on the k3d cluster

1. Clone the project.

   ```bash
   git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/
   ```
2. Build the manager locally and run it on the k3d cluster.

   ```bash
   make -C hack/local run
   ```
> **NOTE:** To clean up the k3d cluster, use the `make k3d stop` make target.

## Manual installation on the k3d cluster

1. Clone the project.

   ```bash
   git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/
   ```

2. Provide the k3d cluster.

   ```bash
   kyma provision k3d
   ```

3. Build and push the Keda Manager image.

   ```bash
   make module-image IMG_REGISTRY=localhost:5001/unsigned/operator-images IMG=localhost:5001/keda-manager-dev-local:0.0.1
   ```

4. Build and push the Keda module.

   ```bash
   make module-build IMG=k3d-kyma-registry:5001/keda-manager-dev-local:0.0.1 MODULE_REGISTRY=localhost:5001/unsigned
   ```

5. Verify if the module and the manager's image are pushed to the local registry.

   ```bash
   curl localhost:5001/v2/_catalog
   ```
You should get a result similar to this example:

   ```json
   {"repositories":["keda-manager-dev-local","unsigned/component-descriptors/kyma.project.io/module/keda"]}
   ```
6. Inspect the generated module template.

> **NOTE:** The following sub-steps are temporary workarounds.

Edit `template.yaml` and:
- change `target` to `control-plane`

   ```yaml
   spec:
    target: control-plane
    ```
> **NOTE:** This is required in the single-cluster mode only.

- change the existing repository context in `spec.descriptor.component`:

   ```yaml
   repositoryContexts:      
     - baseUrl: k3d-kyma-registry.localhost:5000/unsigned
       componentNameMapping: urlPath
       type: ociRegistry
   ```

> **NOTE:** Because Pods inside the k3d cluster use the docker-internal port of the registry, it tries to resolve the registry against port 5000 instead of 5001. K3d has registry aliases, but `module-manager` is not part of k3d and does not know how to properly alias k3d-kyma-registry.localhost:5001.

7. Install the modular Kyma on the k3d cluster.

> **NOTE** This installs the latest versions of `module-manager` and `lifecycle-manager`.

Use the `--template` flag to deploy the Keda module manifest from the beggining, or apply it using kubectl later.

   ```bash
   kyma alpha deploy --template=./template.yaml
   ```
Kyma installation is ready, but the module is not yet activated.

   ```bash
   kubectl get kymas.operator.kyma-project.io -A
   ```
You should get a result similar to the following example:

   ```bash
   NAMESPACE    NAME           STATE   AGE
   kcp-system   default-kyma   Ready   71s
   ```

Keda Module is a known module, but not yet activated.

   ```bash
   kubectl get moduletemplates.operator.kyma-project.io -A 
   ```

You should get a result similar to the following example:

   ```bash
   NAMESPACE    NAME                  AGE
   kcp-system   moduletemplate-keda   2m24s
   ```

8.  Give Module Manager permission to install CRD cluster-wide.

> **NOTE:** `module-manager` must be able to apply CRDs to install modules. In the remote mode (with control-plane managing remote clusters) it gets an administrative kubeconfig, targeting the remote cluster to do so. But in the local mode (single-cluster mode), it uses Service Account and does not have permission to create CRDs by default.

Run the following to make sure the module manager's Service Account gets an administrative role:

   ```bash
   kubectl edit clusterrole module-manager-manager-role
   ```

Add the following element under `rules`:

   ```yaml
   - apiGroups:
     - "*"
     resources:
     - "*"                  
     verbs:                  
     - "*"
  ```

> **NOTE:** This is a temporary workaround and is only required in the single-cluster mode.

9. Enable Keda in the Kyma custom resource (CR).

   ```bash
   kubectl edit kymas.operator.kyma-project.io -n kcp-system default-kyma
   ```

   Add the following field under `spec`:

   ```yaml
     modules:
     - name: keda
       channel: alpha
  ```

## Manual installation

1. Clone the project.

   ```bash
   git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/
   ```

2. Set the `keda-manager` image name.

   ```bash
   export IMG=<DOCKER_USERNAME>/custom-keda-manager:0.0.1
   ```

3. Verify the compatibility.

   ```bash
   make test
   ```
4. Build and push the image to the registry.

   ```bash
   make module-image
   ```

5. Deploy.

   ```bash
   make deploy
   ```

## Using `keda-manager`

- Create a Keda instance.

```bash
kubectl apply -f config/samples/operator_v1alpha1_keda_k3d.yaml
```

- Delete a Keda instance.

```bash
kubectl delete -f config/samples/operator_v1alpha1_keda_k3d.yaml
```

- Update the Keda properties

This example shows how you can modify the Keda docker registry address using the `keda.operator.kyma-project.io` CR

```bash
cat <<EOF | kubectl apply -f -
apiVersion: operator.kyma-project.io/v1alpha1
kind: Keda
metadata:
  name: keda-sample
spec:
  dockerRegistry:
    enableInternal: false
    registryAddress: k3d-kyma-registry:5000
    serverAddress: k3d-kyma-registry:5000
EOF
```
## Troubleshooting

- For MackBook M1 users

Some parts of the scripts may not work because Kyma CLI is not released for Apple Silicon users. To fix it install [Kyma CLI manually](https://github.com/kyma-project/cli#installation) and export the path to it.

   ```bash
   export KYMA=$(which kyma)
   ```

The example error may look like this: `Error: unsupported platform OS_TYPE: Darwin, OS_ARCH: arm64; to mitigate this problem set variable KYMA with the absolute path to kyma-cli binary compatible with your operating system and architecture. Stop.`