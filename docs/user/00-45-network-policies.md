# Network Policies

## Overview

The KEDA module defines network policies to ensure communication within the Kubernetes cluster, particularly in environments where a deny-all network policy is applied.

When a cluster-wide deny-all network policy is enforced, which blocks all ingress and egress traffic by default, the KEDA network policies explicitly allow only the necessary communication paths to ensure the module functions correctly.

## Network Policies

To list the network policies belonging to the KEDA module, run the following command:

```bash
kubectl get networkpolicies -n kyma-system -l kyma-project.io/module=keda
```

The following tables describe the network policies for the KEDA module.

**KEDA Manager Policies**

| Policy Name | Type | Port(s) | Description |
|---|---|---|---|
| `kyma-project.io--keda-manager-allow-to-apiserver` | Egress | 443 (TCP), 6443 (TCP) | Allows egress from the KEDA Manager Pod to the Kubernetes API server. Applied to Pods labeled `app.kubernetes.io/component: keda-manager.kyma-project.io` and `control-plane: manager`. |
| `kyma-project.io--keda-manager-allow-to-dns` | Egress | 53 (TCP/UDP), 8053 (TCP/UDP) | Allows egress from the KEDA Manager Pod to DNS services for cluster and external DNS resolution. Targets any IP on port 53, and Pods labeled `k8s-app: kube-dns` or `k8s-app: node-local-dns` in namespaces labeled `gardener.cloud/purpose: kube-system` on ports 53 and 8053. Applied to Pods labeled `app.kubernetes.io/component: keda-manager.kyma-project.io` and `control-plane: manager`. |

**KEDA Admission Webhooks Policies**

| Policy Name | Type | Port(s) | Description |
|---|---|---|---|
| `kyma-project.io--keda-admission-webhooks-allow-to-apiserver` | Egress | 443 (TCP), 6443 (TCP) | Allows egress from the KEDA Admission Webhooks Pod to the Kubernetes API server. Applied to Pods labeled `app: keda-admission-webhooks`. |
| `kyma-project.io--keda-admission-webhooks-allow-to-dns` | Egress | 53 (TCP/UDP), 8053 (TCP/UDP) | Allows egress from the KEDA Admission Webhooks Pod to DNS services for cluster and external DNS resolution. Targets any IP on port 53, and Pods labeled `k8s-app: kube-dns` or `k8s-app: node-local-dns` in namespaces labeled `gardener.cloud/purpose: kube-system` on ports 53 and 8053. Applied to Pods labeled `app: keda-admission-webhooks`. |
| `kyma-project.io--keda-admission-webhooks-from-apiserver` | Ingress | 9443 (TCP) | Allows ingress to the KEDA Admission Webhooks Pod from any source. This allows the Kubernetes API server to invoke admission webhooks. Applied to Pods labeled `app: keda-admission-webhooks`. |
| `kyma-project.io--keda-admission-webhooks-metrics` | Ingress | 8080 (TCP) | Allows ingress to the metrics endpoint from Pods labeled `networking.kyma-project.io/metrics-scraping: allowed` in the `kyma-system` namespace for metrics scraping. Applied to Pods labeled `app: keda-admission-webhooks`. |

**KEDA Operator Policies**

| Policy Name | Type | Port(s) | Description |
|---|---|---|---|
| `kyma-project.io--keda-operator-allow-to-all` | Egress | All | Allows unrestricted outbound traffic from the KEDA Operator Pod. This is required so the operator can communicate with any service to scrape metrics for scaling purposes. Applied to Pods labeled `app: keda-operator`. |
| `kyma-project.io--keda-operator-allow-from-metrics-apiserver` | Ingress | All | Allows ingress to the KEDA Operator Pod from the KEDA Metrics API Server. Applied to Pods labeled `app: keda-operator`. |
| `kyma-project.io--keda-operator-metrics` | Ingress | 8080 (TCP) | Allows ingress to the metrics endpoint from Pods labeled `networking.kyma-project.io/metrics-scraping: allowed` in the `kyma-system` namespace for metrics scraping. Applied to Pods labeled `app: keda-operator`. |

**KEDA Metrics API Server Policies**

| Policy Name | Type | Port(s) | Description |
|---|---|---|---|
| `kyma-project.io--keda-operator-metrics-apiserver-allow-to-all` | Egress | All | Allows unrestricted outbound traffic from the KEDA Metrics API Server Pod. This is required to allow the metrics API server to communicate with any service to serve metrics for scaling purposes. Applied to Pods labeled `app: keda-operator-metrics-apiserver`. |
| `kyma-project.io--keda-operator-metrics-apiserver-metrics` | Ingress | 8080 (TCP) | Allows ingress to the metrics endpoint from Pods labeled `networking.kyma-project.io/metrics-scraping: allowed` in the `kyma-system` namespace for metrics scraping. Applied to Pods labeled `app: keda-operator-metrics-apiserver`. |
| `kyma-project.io--keda-operator-metrics-apiserver-allow-from-operator` | Ingress | All | Allows ingress to the KEDA Metrics API Server Pod from the KEDA Operator Pod. Applied to Pods labeled `app: keda-operator-metrics-apiserver`. |
| `kyma-project.io--keda-operator-metrics-apiserver-ingress-all-from-apiserver` | Ingress | 6443 (TCP) | Allows ingress to the KEDA Metrics API Server Pod on the HPA-oriented metrics port from any source. This allows the Kubernetes API server to aggregate custom metrics via the metrics API server. Applied to Pods labeled `app: keda-operator-metrics-apiserver`. |
