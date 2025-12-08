# Custom Metric API Example (KEDA)

This example demonstrates scaling a Kubernetes workload using KEDA with a simple custom metric API. The custom metric API serves a static JSON value of 3, which KEDA reads and uses to set the replica count of the scale target to 3 via a managed HPA.

## How it works

- A lightweight NGINX-based server exposes an HTTP endpoint `/metrics` that returns:
  ```
  {"value": 3}
  ```
- KEDA is configured with an `metricAPI` scaler to query that endpoint.
- The scaler uses the returned value to compute the desired replica count.
- KEDA creates and manages an HPA for the `scale-target` Deployment, resulting in 5 replicas (configured maximum).

## Prerequisites

- KEDA installed and running in the cluster.
- Kyma/Istio optional; example includes an annotation to avoid mesh interception of port 80.

## Deploy

Apply the manifests in the k8s folder:

```
kubectl apply -f k8s/
```


Check replicas of the scale target (after KEDA reconciles):

```
kubectl get pods -n keda-custom-metric-api
NAME                                        READY   STATUS    RESTARTS   AGE
custom-metric-api-server-57f77c9b64-f28hs   2/2     Running   0          89s
scale-target-778648fffb-97ln6               2/2     Running   0          7m18s
scale-target-778648fffb-d28xp               2/2     Running   0          17s
scale-target-778648fffb-h8s6t               1/2     Running   0          17s
scale-target-778648fffb-sptn6               2/2     Running   0          47s
scale-target-778648fffb-t2tqr               2/2     Running   0          47s
```

Replicas should be 5, driven by KEDA’s HPA.

```
kubectl get hpa -n keda-custom-metric-api -w
NAME                                REFERENCE                 TARGETS       MINPODS   MAXPODS   REPLICAS   AGE
keda-hpa-custom-metric-api-scaler   Deployment/scale-target   <unknown>/1   1         5         1          5m32s
keda-hpa-custom-metric-api-scaler   Deployment/scale-target   3/1           1         5         1          6m1s
```