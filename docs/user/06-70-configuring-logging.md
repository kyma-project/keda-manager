# Configuring Logging

Learn how to configure logging for Keda Manager and the Keda module components.

## Prerequisites

You have [kubectl](https://kubernetes.io/docs/tasks/tools/) installed.

## Supported Log Levels

From the least to the most verbose: `fatal`, `panic`, `dpanic`, `error`, `warn`, `info` (default), `debug`.

## Supported Log Formats

The supported log formats are the following:

- `json` - Structured JSON format (default)
- `console` (or `text`) - Human-readable console format

## Configure Keda Manager Logging

The Keda manager (`keda-manager`) supports dynamic log-level reconfiguration using a ConfigMap. Changes take effect without requiring a Pod restart.

-  To change the log level, run:

   ```bash
   kubectl patch configmap keda-log-configmap -n kyma-system --type merge -p '{"data":{"log-config.yaml":"logLevel: debug"}}'
   ```

- To change the log level and format, run:

   ```bash
   kubectl patch configmap keda-log-configmap -n kyma-system --type merge -p '{"data":{"log-config.yaml":"logLevel: debug\nlogFormat: json"}}'
   ```

- To verify the change, run:

   ```bash
   kubectl logs -n kyma-system -l app.kubernetes.io/name=keda-manager
   ```

## Configure Keda Module Components Logging

> [!NOTE]
> Applying logging configuration changes to the operator, metrics-server, and admission-webhooks components triggers a restart to apply the new settings.

### Supported Time Encodings

The supported time encodings are the following:

- `rfc3339` - RFC3339 format (default): `2006-01-02T15:04:05Z07:00`
- `rfc3339nano` - RFC3339 with nanoseconds: `2006-01-02T15:04:05.999999999Z07:00`
- `iso8601` - ISO8601 format: `2006-01-02T15:04:05.000Z0700`
- `epoch` - Unix timestamp in seconds
- `millis` - Unix timestamp in milliseconds
- `nano` - Unix timestamp in nanoseconds

### Configuration

Update the Keda custom resource to configure logging for KEDA components:

   ```yaml
   apiVersion: operator.kyma-project.io/v1alpha1
   kind: Keda
   metadata:
     name: default
     namespace: kyma-system
   spec:
     logging:
       operator:
         level: "debug"
         format: "json"
         timeEncoding: "rfc3339"
       metricServer:
         level: "info"
         format: "json"
         timeEncoding: "rfc3339"
       admissionWebhook:
         level: "info"
         format: "json"
         timeEncoding: "rfc3339"
  ```         
> [!NOTE]
> The zap logger used by the Keda module components (operator, metrics-apiserver, admission-webhooks) has fixed field names in JSON format that **cannot be customized**: `level`, `ts`, `logger`, `msg`, `caller`.


### Verify the Changes

- To check the `keda-operator` logs, run:

   ```bash
   kubectl logs -n kyma-system -l app=keda-operator
   ```

- To check the `keda-metrics-apiserver` logs, run:

   ```bash
   kubectl logs -n kyma-system -l app=keda-operator-metrics-apiserver
   ```

- To check the `keda-admission-webhooks` logs, run:

   ```bash
   kubectl logs -n kyma-system -l app=keda-admission-webhooks
   ```

### Limitations

The **klog logs** (Kubernetes API server framework) used by `keda-operator-metrics-apiserver` are always in text format:
   ```
   I0203 12:03:29.253333       1 requestheader_controller.go:180] Starting RequestHeaderAuthRequestController <- Actual log line
   W0203 11:39:56.585970       1 this is a warning example
   E0203 11:39:56.586030       1 this is an error example
   ```
(prefixed with `I`, `W`, `E` for Info, Warning, Error)


## Related Information

For more details about KEDA resources and configuration, see:
- [KEDA Configuration](01-20-configuration.md)
- [KEDA Logs Reference](https://keda.sh/docs/2.18/operate/cluster/#logs)
