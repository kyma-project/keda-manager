# Keda module

## Overview 

Keda module consists of Keda Manager that is an extension to the Kyma runtime. It allows users to install KEDA. It follows the Kubernetes operator pattern to manage the lifecycle of the KEDA installation based on the existence and the content of the dedicated Keda custom resource (CR).

![a](./docs/assets/keda-overview.drawio.svg)

For more information, see [Use Keda Manager to manage KEDA](/docs/keda-management.md).

### What is KEDA?

KEDA is a flexible Event Driven Autoscaler for the Kubernetes workloads. It extends the Kubernetes autoscaling mechanisms with its own metric server and the possibility to make use of external event sources for making scaling decisions. To learn more about KEDA, see the [KEDA documentation](https://keda.sh/docs/latest/concepts/).

## Install Keda module

1. To install Keda Manager manually, apply the following script:

```bash
kubectl create ns kyma-system
kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
```

2. To get KEDA installed, apply the sample Keda CR:

```bash
kubectl apply -f config/samples/operator_v1alpha1_keda_k3d.yaml
```

For more installation options, check [Installation](/docs/contributor/02-01-installation.md).

## Development

For more information about the project structure, make targets, and the CI/CD jobs useful for development, check the [contributor](/docs/contributor/) folder.

## More information

If you want to use Kyma's Keda module, check the [user](/docs/user/) folder to learn more about it. In this folder, you can also find information on how to configure and manage your module. You can also find a demo application that shows how to scale the Kubernetes workloads using Keda API.

The [contributor](/docs/contributor/) folder includes all the necessary information you may need if you want to extend the module with new features. You can learn more about the project structure, make targets, CI/CD jobs that are part of the developing cycle, and different installation options.