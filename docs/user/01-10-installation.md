## Install Keda module

This tutorial shows you how to install the Keda module from the latest release.

1. Clone the project and open the `keda-manager` folder.

   ```bash
   git clone https://github.com/kyma-project/keda-manager.git && cd keda-manager/
   ```
2. To install the Keda module, you must install Keda Manager first. Apply the following script:

   ```bash
   kubectl create ns kyma-system
   kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
   ```

2. To get KEDA installed, apply the sample Keda CR:

   ```bash
   kubectl apply -f config/samples/operator_v1alpha1_keda_k3d.yaml
   ```
You should get a result similar to the this example:

   ```bash
   keda.operator.kyma-project.io/default created
   ```

For more installation options, check [Installation](/docs/contributor/01-10-installation.md).