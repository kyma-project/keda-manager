# Cluster Roles

The Keda module includes several ClusterRoles that are used to manage permissions for the KEDA operator and to aggregate permissions for end users. This document describes all ClusterRoles bundled with the Keda module.

## Keda Edit ClusterRole

The `kyma-keda-edit` ClusterRole allows users to edit KEDA resources.

| API Group | Resources | Verbs |
|-----------|-----------|-------|
| operator.kyma-project.io | kedas | create, delete, get, list, patch, update, watch |
| operator.kyma-project.io | kedas/status | get |
| keda.sh | scaledobjects, scaledjobs, triggerauthentications, clustertriggerauthentications | create, delete, get, list, patch, update, watch |
| keda.sh | scaledobjects/status, scaledjobs/status, triggerauthentications/status, clustertriggerauthentications/status | get |
| eventing.keda.sh | cloudeventsources, clustercloudeventsources | create, delete, get, list, patch, update, watch |
| eventing.keda.sh | cloudeventsources/status, clustercloudeventsources/status | get |

## Keda View ClusterRole

The `kyma-keda-view` ClusterRole allows users to view KEDA resources.

| API Group | Resources | Verbs |
|-----------|-----------|-------|
| operator.kyma-project.io | kedas | get, list, watch |
| operator.kyma-project.io | kedas/status | get |
| keda.sh | scaledobjects, scaledjobs, triggerauthentications, clustertriggerauthentications | get, list, watch |
| keda.sh | scaledobjects/status, scaledjobs/status, triggerauthentications/status, clustertriggerauthentications/status | get |
| eventing.keda.sh | cloudeventsources, clustercloudeventsources | get, list, watch |
| eventing.keda.sh | cloudeventsources/status, clustercloudeventsources/status | get |

## Role Aggregation

The Keda module uses Kubernetes [role aggregation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#aggregated-clusterroles) to automatically extend the default `edit` and `view` ClusterRoles with KEDA-specific permissions.

- **kyma-keda-edit**: Aggregated to `edit` ClusterRole
- **kyma-keda-view**: Aggregated to `view` ClusterRole

This means that users who are granted the default Kubernetes `edit` or `view` ClusterRoles automatically receive the corresponding KEDA permissions without requiring additional role bindings.

