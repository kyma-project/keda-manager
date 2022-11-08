#! /bin/bash

# Add those to /etc/hosts
# 127.0.0.1 k3d-kyma-registry.localhost
# 127.0.0.1 k3d-kyma-registry

# Provision K3D cluster
kyma provision k3d --ci

# Build image of keda manager and  OCI image of keda module
make module-image IMG_REGISTRY=k3d-kyma-registry:5001/unsigned/operator-images
make module-build IMG_REGISTRY=k3d-kyma-registry:5001/unsigned/operator-images MODULE_REGISTRY=k3d-kyma-registry.localhost:5001/unsigned

#Adjust generated module template for single cluster mode on k3d
cat template.yaml | sed -e 's/remote/control-plane/g' -e 's/5001/5000/g'  > template-k3d.yaml

# Install lifecycle and module managers
./bin/kyma-unstable alpha deploy --template=./template-k3d.yaml

# Allow applying CRD to module manager's cluster role
kubectl patch clusterrole module-manager-manager-role --patch-file ./k3d-patches/patch-k3d-module-manager-clusterrole.yaml

# Patch core DNS to enable fetching images from local registry from containers running at k3d 
kubectl patch -n kube-system cm coredns --patch-file ./k3d-patches/patch-k3d-coredns.yaml 

# Enable UI extensibility features in buslao
kubectl apply -f ./k3d-patches/busola-extensibility.yaml
kubectl apply -f ./k3d-patches/busola-kyma-extension.yaml


# Enable Keda
kubectl patch kymas.operator.kyma-project.io -n kcp-system default-kyma --type=merge --patch-file ./k3d-patches/patch-kyma.yaml

# Open busola
# kyma dashboard

# cleanup
# k3d cluster delete kyma
# k3d registry delete k3d-kyma-registry