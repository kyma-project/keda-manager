# Keda Module

## What is KEDA

Kubernetes-based Event Driven Autoscaler [(KEDA)](https://keda.sh/) is an autoscaler that allows you to easily scale your Kubernetes-based resources. You can scale your applications on the basis of the data of your choice.

Keda supports a great number of scalers that help you manage your deployments. For the complete list, check the KEDA [Scalers](https://keda.sh/docs/scalers/) documentation.

For more information about KEDA features, see [KEDA documentation](https://keda.sh/docs).

## Keda module

Keda module is an extension to Kyma that allows you to install and manage KEDA on your Kubernetes cluster, using Keda Manager.
To learn how to enable and disable the Keda module, visit {LINK}.

## Keda Manager

Keda Manager helps you to install and manage KEDA on your cluster. It manages the lifecycle of KEDA based on the dedicated Keda custom resource (CR).

## Useful links
- [KEDA configuration](user/02-01-configuration.md) - provides exemplary configuation of the KEDA components.
- [Use Keda Manager to manage KEDA](contributor/03-01-management.md) - describes how you can manage your KEDA instance using Keda Manager.
- [KEDA demo application](user/06-02-demo-application.md) - shows how to scale the Kubernetes workloads using KEDA API.
- [Installation](contributor/01-01-installation.md) - describes different ways of installing Keda Manager.

For troubleshooting, see:
- [Scripts don't work](contributor/04-01-scripts-not-working.md)