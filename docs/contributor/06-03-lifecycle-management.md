# Lifecycle management of Keda Manager in Kyma

When you enable the Keda module using your Kyma runtime Kyma custom resource (CR), the Lifecycle Manager downloads the bundled package of the Keda Manager and installs it. Additionally, it applies a sample Keda CR, which triggers Keda Manager to install the Keda module.

![a](/docs/assets/keda-lm-overview.drawio.svg)