# ClusterRoles

The Keda module includes several ClusterRoles that are used to manage permissions for the KEDA operator and to aggregate permissions for end users. This document describes all ClusterRoles bundled with the Keda module.

## Keda Edit ClusterRole

With the `kyma-keda-edit` ClusterRole, you can edit the KEDA resources. For the available options, see the following table:

| API Group | Resources | Verbs |
|-----------|-----------|-------|
| `operator.kyma-project.io` | `kedas` | `create, delete, get, list, patch, update, watch` |
| `operator.kyma-project.io` | `kedas/status` | `get` |
| `keda.sh` | `scaledobjects, scaledjobs, triggerauthentications, clustertriggerauthentications` | `create, delete, get, list, patch, update, watch` |
| `keda.sh` | `scaledobjects/status, scaledjobs/status, triggerauthentications/status, clustertriggerauthentications/status` | `get` |
| `eventing.keda.sh` | `cloudeventsources, clustercloudeventsources` | `create, delete, get, list, patch, update, watch` |
| `eventing.keda.sh` | `cloudeventsources/status, clustercloudeventsources/status` | `get` |

## Keda View ClusterRole

With the `kyma-keda-view` ClusterRole, you can view KEDA resources. For the available options, see the following table:

| API Group | Resources | Verbs |
|-----------|-----------|-------|
| `operator.kyma-project.io` | `kedas` | `get, list, watch` |
| `operator.kyma-project.io` | `kedas/status` | `get` |
| `keda.sh` | `scaledobjects, scaledjobs, triggerauthentications, clustertriggerauthentications` | `get, list, watch` |
| `keda.sh` | `scaledobjects/status, scaledjobs/status, triggerauthentications/status, clustertriggerauthentications/status` | `get` |
| `eventing.keda.sh` | `cloudeventsources, clustercloudeventsources` | `get, list, watch` |
| `eventing.keda.sh` | `cloudeventsources/status, clustercloudeventsources/status` | `get` |

## Role Aggregation

The Keda module uses the Kubernetes [role aggregation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#aggregated-clusterroles) to automatically extend the default `edit` and `view` ClusterRoles with KEDA-specific permissions.

- **kyma-keda-edit**: Aggregated to `edit` ClusterRole
- **kyma-keda-view**: Aggregated to `view` ClusterRole

This means that if you have the default Kubernetes `edit` or `view` ClusterRoles, you automatically receive the corresponding KEDA permissions without requiring additional role bindings.

