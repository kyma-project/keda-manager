# Keda Manager

## Status
![GitHub tag checks state](https://img.shields.io/github/checks-status/kyma-project/keda-manager/main?label=keda-operator&link=https%3A%2F%2Fgithub.com%2Fkyma-project%2Fkeda-manager%2Fcommits%2Fmain)

## Overview 

Keda Manager is an extension to the Kyma runtime. It allows users to install KEDA. It follows the Kubernetes operator pattern to manage the lifecycle of the KEDA installation based on the existence and the content of the dedicated Keda custom resource (CR).

![Keda manager overview](docs/assets/keda-overview.drawio.svg)

1. User applies the Keda CR.
2. Keda Manager watches the Keda CR.
3. Keda Manager reconciles the KEDA workloads.

For more information, see [Use Keda Manager to manage KEDA](docs/contributor/02-10-management.md).

## What is KEDA?

KEDA is a flexible Event Driven Autoscaler for the Kubernetes workloads. It extends the Kubernetes autoscaling mechanisms with its own metric server and the possibility to make use of external event sources for making scaling decisions. To learn more about KEDA, see the [KEDA documentation](https://keda.sh/docs/latest/concepts/).

## Install Keda Manager and KEDA from the latest release

### Prerequisites

- Access to a Kubernetes (v1.24 or higher) cluster
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

### Procedure


1. To install KEDA, you must install Keda Manager first. Apply the following script:

   ```bash
   kubectl create ns kyma-system
   kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
   ```

2. To get KEDA installed, apply the sample Keda CR:

   ```bash
   kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-default-cr.yaml
   ```
   You should get a result similar to this example:

   ```bash
   keda.operator.kyma-project.io/default created
   ```

For more installation options, visit [Install Keda Manager](/docs/contributor/01-10-installation.md)

## Read more

If you want to use Kyma's Keda module, check the [user](/docs/user/) folder to learn more about it. In this folder, you also get information on how to configure your module. You also find a demo application that shows how to scale the Kubernetes workloads using Keda API.

The [contributor](/docs/contributor/) folder includes all the necessary information on how to extend the module with new features. You can learn more about the project structure, make targets, CI/CD jobs that are part of the developing cycle, and different installation options.
