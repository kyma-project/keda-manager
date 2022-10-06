# Keda Manager

## Overview

Keda Manager is a module compatible with `lifecycle-manager` that allows to add KEDA Event Driven Autoscaler to Kyma ecosystem.

See also:
- [lifecycle-manager documetation](https://github.com/kyma-project/lifecycle-manager)
- [KEDA documentation](https://keda.sh/docs/2.7/concepts/)

## Prerequisites

- access to a k8s cluster
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

## Manual installation

This section is intended mainly for contributors.

### Building project

```bash
make build
```

### Build image

```bash
make docker-build IMG=<image-name>:<image-tag>
```

### Push image to registry 

If you are using k3d.

```bash
k3d image import <image-name>:>image-tag> -c <k3d_cluster>
```

If using globally available docker registry

```bash
make docker-push IMG=<image-name>:<image-tag>
```

### Deploy

```bash
make deploy IMG=<image-name>:<image-tag>
```

### Apply sample

```bash
kubectl apply -f config/samples/operator_v1alpha1_keda.yaml
```

