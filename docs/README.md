# Docs

## Overview

Keda Manager is an extension to the Kyma ecosystem that allows users to install KEDA. It follows the Kubernetes operator pattern to manage the lifecycle of the KEDA installation based on the existence and the content of the dedicated Keda custom resource (CR).

![a](./docs/assets/keda-overview.drawio.svg)

### What is KEDA?

KEDA is a flexible Event Driven Autoscaler for the Kubernetes workloads. It extends the Kubernetes autoscaling mechanisms with its own metric server and the possibility to make use of external event sources to make scaling decisions. To learn more about KEDA, see [KEDA documentation](https://keda.sh/docs/latest/concepts/).

## Installation

To learn how to install Keda Manager locally on k3d, visit the [Local k3d setup](installation-on-k3d.md) tutorial.

For more installation options, visit the [Install](../README.md#install) section.

## Configuration

For Keda Manager configuration options, see [Keda configuration](keda-configuration.md).

## Demo 

Visit [Keda Demo Application](keda-demo-application.md) to see how to scale the Kubernetes workloads using Keda API based on a simple CPU consumption case.
