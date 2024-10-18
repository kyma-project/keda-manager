# Keda Module Configuration

This document describes how to configure the Keda module using the Keda CustomResourceDefinition (CRD).
See how to configure the **logging.level** or resource consumption.

- You can change the logging level of the KEDA workloads. To change **logging.level**, choose one of the accepted values:
   - `debug` - is the most detailed option. Useful for a developer during debugging.
   - `info` - provides standard log level indicating operations within the Keda module. For example, it can show whether the workload scaling operation was successful or not.
   - `error` - shows error logs only. This means only log messages corresponding to errors and misconfigurations are visible in logs.

   ```yaml
   spec:
     logging:
       operator:
         level: "debug"
   ```

- To enable the Istio sidecar injection for **operator** and **metricServer**, set the value of **enabledSidecarInjection** to `true`. For example:

  ```yaml
  spec:
    istio:
      metricServer:
        enabledSidecarInjection: true
      operator:
        enabledSidecarInjection: true
  ```

- To change the resource consumption, enter your preferred values for **operator**, **metricServer** and **admissionWebhook**. For example:

   ```yaml
   spec:
     resources:
       operator:
         limits:
           cpu: "1"
           memory: "200Mi"
         requests:
           cpu: "150m"
           memory: "150Mi"
       metricServer:
         limits:
           cpu: "1"
           memory: "1000Mi"
         requests:
           cpu: "150m"
           memory: "500Mi"
       admissionWebhook:
         limits:
           cpu: "1"
           memory: "1000Mi"
         requests:
           cpu: "50m"
           memory: "800Mi"
   
   ```

For more information about the Keda resources, visit the [Keda concepts](https://keda.sh/docs/latest/concepts/) documentation.
