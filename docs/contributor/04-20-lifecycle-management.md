# Lifecycle Management of Keda Manager in Kyma

When you add the Keda module using your Kyma custom resource (CR), Kyma Lifecycle Manager (KLM) downloads the bundled package of Keda Manager and installs it. Additionally, it applies a sample Keda CR, which triggers Keda Manager to install the Keda module.

![Add Keda module with LM](../assets/keda-lm-overview.drawio.svg)

1. User adds the Keda module in the Kyma CR.
2. KLM reads the module template of the Keda module.
3. KLM deploys Keda Manager, using artifacts from the module template.
4. KLM applies the default Keda CR from the module template.
5. Keda Manager watches the Keda CR.
6. Keda Manager reconciles the KEDA workloads.
7. User can configure the Keda module by changing the Keda CR **spec**. Keda Manager reconciles the workloads accordingly.
