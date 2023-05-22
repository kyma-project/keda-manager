# Install Keda Module

## Install Keda module manually

1. Apply the following script to install Keda Manager:

   ```bash
   kubectl create ns kyma-system
   kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
   ```

2. To get KEDA installed, apply the sample Keda CR:

   ```bash
   kubectl apply -f config/samples/operator_v1alpha1_keda_k3d.yaml
   ```

## Install on Kyma runtime

This section describes how to set up the Keda module (KEDA + Keda Manager) on top of the Kyma installation with Lifecycle Manager.
In such a setup, you don't need to install Keda Manager. It is installed and managed by Lifecycle Manager.

### Lifecycle management of Keda Manager in Kyma

When you enable the Keda module using your Kyma runtime Kyma custom resource (CR), the Lifecycle Manager downloads the bundled package of the Keda Manager and installs it. Additionally, it applies a sample Keda CR, which triggers Keda Manager to install the Keda module.

![a](assets/keda-lm-overview.drawio.svg)

To enable the Keda module run:

   ```bash
   kyma alpha enable module keda -c fast
   ```