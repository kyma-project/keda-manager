# Keda Manager

## Overview 

Keda Manager is an extension to the Kyma ecosystem that allows users to install KEDA. It follows the Kubernetes operator pattern to manage the lifecycle of the KEDA installation based on the existence and the content of the dedicated Keda custom resource (CR).

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

1. To get KEDA installed, apply the sample Keda CR:

```bash
kubectl apply -f config/samples/operator_v1alpha1_keda_k3d.yaml
```

For more installation options, check the [Install Keda Manager](/docs/keda-installation.md) tutorial.

## User interface

Keda Manager is not only an API extension to the Kyma ecosystem, but it also extends the UI of the Kyma Dashboard.
It uses the [UI extensibility](https://github.com/kyma-project/busola/tree/main/docs/extensibility) feature of Kyma Dashboard.
In the [ui-extensions](config/ui-extensions) folder, you can find configuration for the UI components (for example, `list` view, `form` view, `details` view) that helps you manipulate with Keda CRs - `ScaledObjects`.
This configuration is applied as part of the Keda Manager resources. Thanks to that, it comes and goes depending on whether the Keda module is enabled or disabled.

## Releasing new versions 

The release of a new version of the Keda module is realized using the release channels.
This means that new versions are submitted to a given channel.

Current versions per each channel are represented by the [ModuleTemplate CR](https://github.com/kyma-project/lifecycle-manager/blob/main/docs/technical-reference/api/moduleTemplate-cr.md) submitted to a matching folder in the Kyma Git repository.

Having merged all the changes into the `main` branch in the `keda-manager` repository, the [CI/CD jobs](/docs/CI-CD-description.md) will bundle module images and generate ModuleTemplate for you.
Submit your ModuleTemplate into the desired channel using a pull request to the Kyma repository.
A series of governance jobs will start testing if the new candidate version fulfills the criteria described in the [module submission process](https://github.com/kyma-project/community/tree/main/concepts/modularization#module-submission-process).


## Keda module footprint

This section describes the impact the installed Keda module has on the cluster resources.

TBD