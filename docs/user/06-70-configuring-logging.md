# Configuring Logging

This document describes how to configure logging for the KEDA module components. The module consists of:
- **KEDA Manager** - Supports dynamic log reconfiguration without restart
- **KEDA Components** (operator, metrics-apiserver, admission-webhooks) - Require pod restart for logging changes

## Supported Log Levels

From the least to the most verbose: `fatal`, `panic`, `dpanic`, `error`, `warn`, `info` (default), `debug`.

## Supported Log Formats

- `json` - Structured JSON format (default)
- `console` (or `text`) - Human-readable console format

## Configure KEDA Manager Logging

The KEDA manager (keda-manager) supports **dynamic log level reconfiguration** through a ConfigMap. Changes take effect without requiring a pod restart.

   ```bash
   # Change log level only
   kubectl patch configmap keda-log-configmap -n kyma-system --type merge -p '{"data":{"log-config.yaml":"logLevel: debug"}}'

   # Change both level and format
   kubectl patch configmap keda-log-configmap -n kyma-system --type merge -p '{"data":{"log-config.yaml":"logLevel: debug\nlogFormat: json"}}'
   ```

Verify the change :

   ```bash
   kubectl logs -n kyma-system -l app.kubernetes.io/name=keda-manager
   ```

## Configure KEDA Components Logging (Requires Restart)

> [NOTE]
> Logging configuration changes for KEDA components (operator, metrics-apiserver, admission-webhooks) will trigger a restart to apply the new settings.

### Supported Time Encodings

- `rfc3339` - RFC3339 format (default): `2006-01-02T15:04:05Z07:00`
- `rfc3339nano` - RFC3339 with nanoseconds: `2006-01-02T15:04:05.999999999Z07:00`
- `iso8601` - ISO8601 format: `2006-01-02T15:04:05.000Z0700`
- `epoch` - Unix timestamp in seconds
- `millis` - Unix timestamp in milliseconds
- `nano` - Unix timestamp in nanoseconds

### Configuration

Update the Keda Custom Resource to configure logging for KEDA components:

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
>[NOTE]
>The zap logger used by KEDA components (operator, metrics-apiserver, admission-webhooks) has fixed field names in JSON format that **cannot be customized**: `level`, `ts`, `logger`, `msg`, `caller`.


### Verify the Changes

   ```bash
   # Check keda-operator logs
   kubectl logs -n kyma-system -l app=keda-operator

   # Check keda-metrics-apiserver logs
   kubectl logs -n kyma-system -l app=keda-operator-metrics-apiserver

   # Check keda-admission-webhooks logs
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
