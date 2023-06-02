# Extend user interface (UI)

Keda Manager is not only an API extension to the Kyma runtime, but you can also use it to configure a dedicated UI for your CustomResourceDefinition (CRD).
To do that, use the [UI extensibility](https://github.com/kyma-project/busola/tree/main/docs/extensibility) feature of Kyma Dashboard.
In the [ui-extensions](/config/ui-extensions/) folder, you can find configuration for the UI components (for example, the `list`, `form`, or `details` views) that allows you to create a dedicated UI page for your Keda CR - `ScaledObjects`.
This configuration is applied as part of the Keda Manager resources. Thanks to that, it comes and goes depending on whether the Keda module is enabled or disabled.