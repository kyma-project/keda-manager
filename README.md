# Keda Manager

## Overview

Keda Manager is a module compatible with Lifecycle Manager that allows you to add KEDA Event Driven Autoscaler to the Kyma ecosystem. 

See also:
- [lifecycle-manager documetation](https://github.com/kyma-project/lifecycle-manager)
- [KEDA documentation](https://keda.sh/docs/2.7/concepts/)

## Prerequisites

- Access to a k8s cluster
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)

```bash
# you could use one of the following options

# option 1: using brew
brew install kubebuilder

# option 2: fetch sources directly
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
```

## Manual `keda-manager` installation


1. Clone project

```bash
git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/operator
```

2. Set `keda-manager` image name

```bash
export IMG=custom-keda-manager:0.0.1
export K3D_CLUSTER_NAME=keda-manager-demo
```

3. Build project

```bash
make build
```

4. Build image

```bash
make docker-build
```

5. Push image to registry

<div tabs name="Push image" group="keda-installation">
  <details>
  <summary label="k3d">
  k3d
  </summary>

   ```bash
   k3d image import $IMG -c $K3D_CLUSTER_NAME
   ```
  </details>
  <details>
  <summary label="Docker registry">
  Globally available Docker registry
  </summary>

   ```bash
   make docker-push
   ```

  </details>
</div>

### Deploy

```bash
make deploy
```

## Using `keda-manager`

- Create Keda instance

```bash
kubectl apply -f config/samples/operator_v1alpha1_keda.yaml
```

- Delete Keda instance

```bash
kubectl delete -f config/samples/operator_v1alpha1_keda.yaml
```

- Update Keda properties

This example shows how you can modify the Keda logging level using the `keda.operator.kyma-project.io` CR

```bash
cat <<EOF | kubectl apply -f -
apiVersion: operator.kyma-project.io/v1alpha1
kind: Keda
metadata:
  name: keda-sample
spec:
  logging:
    operator:
      level: "info"
EOF
```

