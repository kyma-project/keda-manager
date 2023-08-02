# Lifecycle management of Keda Manager in Kyma

When you enable the Keda module using your Kyma custom resource (CR), the Lifecycle Manager (LM) downloads the bundled package of the Keda Manager and installs it. Additionally, it applies a sample Keda CR, which triggers Keda Manager to install the Keda module.

![Enable Keda module with LM](../assets/keda-lm-overview.drawio.svg)

1. User enables the Keda module in the Kyma CR.
2. Lifecycle Manager reads the module template of the Keda module.
3. Lifecycle Manager deploys Keda Manager, using artifacts from the module template.
4. Lifecycle Manager applies the default Keda CR from the module template.
5. Keda Manager watches the Keda CR.
6. Keda Manager reconciles the KEDA workloads.
7. User can configure the Keda module by changing the Keda CR **spec**. Keda Manager reconciles the workloads accordingly.