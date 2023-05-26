# Keda Module

## What is KEDA

Kubernetes-based Event Driven Autoscaler [(KEDA)](https://keda.sh/) is an autoscaler that allows you to scale easily your Kubernetes-based resources. You can scale your applications on the basis of data of your choice.

Keda supports a great number of scalers that help you manage your deployments. For the complete list, check the KEDA [Scalers](https://keda.sh/docs/scalers/) documentation.

For more information about KEDA features, see [KEDA documentation](https://keda.sh/docs).

## Keda module

Keda module is a solution introduced in Kyma that allows you to install and manage KEDA on your Kubernetes cluster, using Keda Manager.
To learn how to enable and disable the Keda module, visit {LINK}.

## Keda Manager

Keda Manager helps you to install and manage KEDA on your cluster. It manages the lifecycle of KEDA based on the dedicated Keda custom resource (CR).

## User interface (UI)

Keda Manager is not only an API extension to the Kyma runtime, but you can also use it to configure a dedicated UI for your CustomResourceDefinition (CRD).
To do that, use the [UI extensibility](https://github.com/kyma-project/busola/tree/main/docs/extensibility) feature of Kyma Dashboard.
In the [ui-extensions](/config/ui-extensions/) folder, you can find configuration for the UI components (for example, the `list`, `form`, or `details` views) that allows you to create a dedicated UI page for your Keda CR - `ScaledObjects`.
This configuration is applied as part of the Keda Manager resources. Thanks to that, it comes and goes depending on whether the Keda module is enabled or disabled.

## Useful links
- [KEDA configuration](02-01-keda-configuration.md) - provides exemplary configuation of the KEDA components.
- [Use Keda Manager to manage KEDA](02-02-keda-management.md) - describes how you can manage your KEDA instance using Keda Manager.
- [KEDA Demo Application](03-01-keda-demo-application.md) - shows how to scale the Kubernetes workloads using KEDA API.
- [Installation](/docs/contributor/02-01-installation.md) - describes different ways of installing Keda Manager.

For troubleshooting, see:
- [Scripts don't work](04-01-scripts-not-working.md)
