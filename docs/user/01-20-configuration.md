# Keda configuration

This document describes how to configure the Keda module using Kyma CustomResourceDefinition (CRD).
See how to configure the **logging.level** or resource consumption.

- To change the logging level, choose one of the accepted values: `debug`, `info`, or `error`. For example:

   ```yaml
   spec:
     logging:
       operator:
         level: "debug"
   ```
- Change the operator and metricServer resource consumption using your preferred values. For example:

   ```yaml
     resources:
       operator:
         limits:
           cpu: "1"
           memory: "200Mi"
         requests:
           cpu: "0.5"
           memory: "150Mi"
       metricServer:
         limits:
           cpu: "1"
           memory: "1000Mi"
         requests:
           cpu: "300m"
           memory: "500Mi"
   ```
For more information about the Keda resources, visit the [Keda concepts](https://keda.sh/docs/latest/concepts/) documentation.