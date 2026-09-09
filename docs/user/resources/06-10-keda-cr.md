# Keda

The `kedas.operator.kyma-project.io` CustomResourceDefinition (CRD) is a detailed description of the Keda module configuration that you want to install on your cluster. To get the up-to-date CRD and show the output in the YAML format, run this command:

   ```bash
   kubectl get crd kedas.operator.kyma-project.io -o yaml
   ```

## Sample Custom Resource

The following Keda custom resource (CR) shows configuration of the Keda module with custom logging settings and resource limits.

   ```yaml
   apiVersion: operator.kyma-project.io/v1alpha1
   kind: Keda
   metadata:
     finalizers:
     - keda-manager.kyma-project.io/deletion-hook
     name: default
     namespace: kyma-system
   spec:
     logging:
       operator:
         level: "info"
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
     resources:
       operator:
         limits:
           cpu: "800m"
           memory: "800Mi"
         requests:
           cpu: "50m"
           memory: "100Mi"
       metricServer:
         limits:
           cpu: "800m"
           memory: "800Mi"
         requests:
           cpu: "50m"
           memory: "100Mi"
       admissionWebhook:
         limits:
           cpu: "800m"
           memory: "800Mi"
         requests:
           cpu: "50m"
           memory: "100Mi"
   status:
     conditions:
     - lastTransitionTime: "2024-01-15T10:00:00Z"
       message: Keda installed
       reason: Verified
       status: "True"
       type: Installed
     kedaVersion: 2.16.0
     served: "True"
     state: Ready
   ```

## Custom Resource Parameters

For details, see the [Keda specification file](https://github.com/kyma-project/keda-manager/blob/main/api/v1alpha1/keda_types.go).
<!-- TABLE-START -->
<!-- markdownlint-disable-next-line -->
### keda.operator.kyma-project.io/v1alpha1

**Spec:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| **istio** | object | Istio sidecar injection configuration. |
| **istio.&#x200b;operator** | object | Istio configuration for the KEDA operator component. |
| **istio.&#x200b;operator.&#x200b;enabledSidecarInjection** | boolean | Enables Istio sidecar injection for the KEDA operator Pod. |
| **istio.&#x200b;metricServer** | object | Istio configuration for the KEDA metrics server component. |
| **istio.&#x200b;metricServer.&#x200b;enabledSidecarInjection** | boolean | Enables Istio sidecar injection for the KEDA metrics server Pod. |
| **logging** | object | Logging configuration for KEDA components. |
| **logging.&#x200b;operator** | object | Logging configuration for the KEDA operator component. |
| **logging.&#x200b;operator.&#x200b;level** | string | Log level. The allowed values are `debug`, `info`, and `error`. The default value is `info`. |
| **logging.&#x200b;operator.&#x200b;format** | string | Log format. The allowed values are `json`, `text`, and `console`. The default value is `console`. |
| **logging.&#x200b;operator.&#x200b;timeEncoding** | string | Log timestamp format. The allowed values are `epoch`, `millis`, `nano`, `iso8601`, `rfc3339`, and `rfc3339nano`. The default value is `rfc3339`. |
| **logging.&#x200b;metricServer** | object | Logging configuration for the KEDA metrics server component. |
| **logging.&#x200b;metricServer.&#x200b;level** | string | Log level. The allowed values are `debug`, `info`, and `error`. The default value is `info`. |
| **logging.&#x200b;metricServer.&#x200b;format** | string | Log format. The allowed values are `json`, `text`, and `console`. The default value is `console`. |
| **logging.&#x200b;metricServer.&#x200b;timeEncoding** | string | Log timestamp format. The allowed values are `epoch`, `millis`, `nano`, `iso8601`, `rfc3339`, and `rfc3339nano`. The default value is `rfc3339`. |
| **logging.&#x200b;admissionWebhook** | object | Logging configuration for the KEDA admission webhook component. |
| **logging.&#x200b;admissionWebhook.&#x200b;level** | string | Log level. The allowed values are `debug`, `info`, and `error`. The default value is `info`. |
| **logging.&#x200b;admissionWebhook.&#x200b;format** | string | Log format. The allowed values are `json`, `text`, and `console`. The default value is `console`. |
| **logging.&#x200b;admissionWebhook.&#x200b;timeEncoding** | string | Log timestamp format. The allowed values are `epoch`, `millis`, `nano`, `iso8601`, `rfc3339`, and `rfc3339nano`. The default value is `rfc3339`. |
| **resources** | object | CPU and memory resource requirements for KEDA components. |
| **resources.&#x200b;operator** | object | Resource requirements for the KEDA operator component. |
| **resources.&#x200b;operator.&#x200b;limits.&#x200b;cpu** | string | CPU limit for the KEDA operator. |
| **resources.&#x200b;operator.&#x200b;limits.&#x200b;memory** | string | Memory limit for the KEDA operator. |
| **resources.&#x200b;operator.&#x200b;requests.&#x200b;cpu** | string | CPU request for the KEDA operator. |
| **resources.&#x200b;operator.&#x200b;requests.&#x200b;memory** | string | Memory request for the KEDA operator. |
| **resources.&#x200b;metricServer** | object | Resource requirements for the KEDA metrics server component. |
| **resources.&#x200b;metricServer.&#x200b;limits.&#x200b;cpu** | string | CPU limit for the KEDA metrics server. |
| **resources.&#x200b;metricServer.&#x200b;limits.&#x200b;memory** | string | Memory limit for the KEDA metrics server. |
| **resources.&#x200b;metricServer.&#x200b;requests.&#x200b;cpu** | string | CPU request for the KEDA metrics server. |
| **resources.&#x200b;metricServer.&#x200b;requests.&#x200b;memory** | string | Memory request for the KEDA metrics server. |
| **resources.&#x200b;admissionWebhook** | object | Resource requirements for the KEDA admission webhook component. |
| **resources.&#x200b;admissionWebhook.&#x200b;limits.&#x200b;cpu** | string | CPU limit for the KEDA admission webhook. |
| **resources.&#x200b;admissionWebhook.&#x200b;limits.&#x200b;memory** | string | Memory limit for the KEDA admission webhook. |
| **resources.&#x200b;admissionWebhook.&#x200b;requests.&#x200b;cpu** | string | CPU request for the KEDA admission webhook. |
| **resources.&#x200b;admissionWebhook.&#x200b;requests.&#x200b;memory** | string | Memory request for the KEDA admission webhook. |
| **env** | \[\]object | List of environment variables to set in KEDA components. Follows the Kubernetes `EnvVar` specification. |
| **env[].&#x200b;name** (required) | string | Name of the environment variable. |
| **env[].&#x200b;value** | string | Value of the environment variable. |
| **env[].&#x200b;valueFrom** | object | Source for the environment variable value, such as a ConfigMap or Secret reference. |
| **podAnnotations** | object | Annotations to add to the Pods of KEDA components. |
| **podAnnotations.&#x200b;operator** | map | Annotations for the KEDA operator Pods. |
| **podAnnotations.&#x200b;metricServer** | map | Annotations for the KEDA metrics server Pods. |
| **podAnnotations.&#x200b;admissionWebhook** | map | Annotations for the KEDA admission webhook Pods. |

**Status:**

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| **state** | string | Signifies the current state of the Keda module. The value can be one of `Ready`, `Processing`, `Error`, `Warning`, or `Deleting`. |
| **served** (required) | string | Signifies that the current Keda CR is managed. The value can be `True` or `False`. |
| **kedaVersion** | string | Version of the installed KEDA. |
| **conditions** | \[\]object | Conditions associated with the Keda CR status. |
| **conditions.&#x200b;lastTransitionTime** (required) | string | Specifies the last time the condition transitioned from one status to another. |
| **conditions.&#x200b;message** (required) | string | Provides a human-readable message indicating details about the transition. |
| **conditions.&#x200b;observedGeneration** | integer | Represents the **.metadata.generation** that the condition was set based upon. |
| **conditions.&#x200b;reason** (required) | string | Contains a programmatic identifier indicating the reason for the condition's last transition. |
| **conditions.&#x200b;status** (required) | string | Specifies the status of the condition. The value is either `True`, `False`, or `Unknown`. |
| **conditions.&#x200b;type** (required) | string | Specifies the condition type in camelCase. |

<!-- TABLE-END -->

## Keda CR Conditions

This section describes the possible states of the Keda CR. The following condition types are used: `Installed`, `DeploymentFailure`, and `Deleted`.

| No | CR State   | Condition type    | Condition status | Condition reason         | Remark                                      |
|----|------------|-------------------|------------------|--------------------------|---------------------------------------------|
| 1  | Ready      | Installed         | true             | Verified                 | Server ready                                |
| 2  | Processing | Installed         | unknown          | Initialized              | Initialized                                 |
| 3  | Processing | Installed         | unknown          | Verification             | Verification in progress                    |
| 4  | Error      | Installed         | false            | ApplyObjError            | Apply object error                          |
| 5  | Error      | Installed         | false            | KedaDeploymentUpdateErr  | Deployment update error                     |
| 6  | Error      | Installed         | false            | VerificationErr          | Verification error                          |
| 7  | Error      | Installed         | false            | KedaDuplicated           | Only one instance of Keda is allowed        |
| 8  | Error      | DeploymentFailure | true             | DeploymentReplicaFailure | Workloads have the ReplicaFailure condition |
| 9  | Deleting   | Deleted           | unknown          | Deletion                 | Deletion in progress                        |
| 10 | Deleting   | Deleted           | true             | Deleted                  | Keda module deleted                         |
| 11 | Error      | Deleted           | false            | DeletionErr              | Deletion failed                             |
| 12 | Error      | Installed         | false            | ValidationErr            | Validation error                            |
