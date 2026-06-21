# Scale to Zero with KEDA HTTP Add-on

## Overview

This example demonstrates how to use the [KEDA HTTP Add-on](https://github.com/kedacore/http-add-on) on a Kyma cluster to achieve HTTP-based scale-to-zero and scale-from-zero for workloads, without losing any requests.
It uses:

- [KEDA HTTP Add-on](https://github.com/kedacore/http-add-on) to intercept, queue, and count incoming HTTP requests — enabling scale-to-zero and scale-from-zero without lost requests,
- [KEDA](https://keda.sh/) to drive the workload's scaling based on request rate metrics provided by the HTTP Add-on,
- A demo application that returns request-specific information (Pod name, timestamp, request body) to verify that no requests are lost during scaling,
- Istio service mesh to provide mTLS encryption between all components and to expose the application via API Gateway,
- API Gateway (APIRule v2) to route external HTTPS traffic to the HTTP Add-on's Interceptor.

## Prerequisites

- Kyma as the target Kubernetes runtime.
- [Keda, Istio and API Gateway modules installed](https://kyma-project.io/02-get-started/01-quick-install.html#steps)

## Procedure

### 1. Enable the HTTP Add-on via Annotations

Enable the HTTP Add-on by annotating the Keda custom resource (CR). The Keda Manager automatically installs the add-on in the specified namespace with Istio sidecar injection and the required port exclusion configured:

```bash
kubectl annotate keda -n kyma-system default \
  keda.kyma-project.io/addon-enabled=true \
  keda.kyma-project.io/addon-version=<addon-version> \
  keda.kyma-project.io/addon-namespace=<addon-namespace>
```

### 2. Enable Istio Sidecar Injection for the Add-on

Annotate the Keda CR to enable Istio sidecar injection on the HTTP Add-on Deployments:

```bash
kubectl annotate keda -n kyma-system default \
  keda.kyma-project.io/addon-istio-injection=true
```

### 3. Wait for the Add-on to Be Ready

Verify that the add-on condition is `True`:

```bash
kubectl get keda -n kyma-system default -o jsonpath='{.status.conditions[?(@.type=="Addon")].status}'
```

Expected output: `True`

Verify that all add-on Pods are running with Istio sidecar (2/2):

```bash
kubectl get pods -n kyma-system
NAME                                               READY   STATUS    RESTARTS   AGE
keda-add-ons-http-interceptor-b98dc64f9-gswbn      2/2     Running   0          2m
keda-add-ons-http-operator-66bb6c6f8b-qwrm6        2/2     Running   0          2m
keda-add-ons-http-scaler-5cd8bd8499-gs8wh          2/2     Running   0          2m
```

### 4. Get Your Cluster Domain

```bash
export DOMAIN=$(kubectl get configmap shoot-info -n kube-system -o jsonpath='{.data.domain}')
echo $DOMAIN
```

### 5. Edit the Example Resources

Edit the following files and replace `<YOUR_DOMAIN>` with your cluster domain:

- `k8s-resources/httpscaledobject.yaml` — set the host
- `k8s-resources/envoyfilter.yaml` — set the vhost name (format: `http-echo-keda.<YOUR_DOMAIN>:443`)

### 6. Apply the Example Resources

```bash
kubectl create namespace demo-app
kubectl label namespace demo-app istio-injection=enabled
kubectl apply -f ./k8s-resources
```

### Key Resources

| Resource | Kind | Description |
|---|---|---|
| `apirule.yaml` | `APIRule` | Routes external HTTPS traffic from `https://http-echo-keda.<YOUR_DOMAIN>` through Istio Ingress Gateway to the HTTP Add-on Interceptor's Service (`keda-add-ons-http-interceptor-proxy`) in the `keda` namespace. The Interceptor handles request queuing and forwarding to the application. |
| `httpscaledobject.yaml` | `HTTPScaledObject` | Configures the HTTP Add-on to scale the `http-echo` Deployment based on incoming request rate. Sets `min: 0` to allow scale-to-zero and `max: 10` for scale-out. The operator automatically creates a corresponding KEDA `ScaledObject`. |
| `envoyfilter.yaml` | `EnvoyFilter` | Configures the Istio Ingress Gateway to automatically retry requests that fail with `5xx`, `connect-failure`, or `reset` — up to 100 times with a 3-second per-try timeout. This is critical for cold start: when the application is scaling from zero, the first forwarded request may arrive before the Pod is ready. The retry policy ensures the request is eventually delivered without returning an error to the client. |
| `demo-app.yaml` | `Deployment` + `Service` | Deploys a lightweight Node.js HTTP server that returns a JSON response with the handling Pod's name, a timestamp, and the request body. This allows you to verify that every request was processed and no request was lost during scale-from-zero. |

## Test the Application

Initially, the application Pod is scaled down to zero.

1. List HPA for the demo application and check that the current replica count is zero:

```bash
kubectl get hpa -n demo-app
NAME                 REFERENCE              TARGETS              MINPODS   MAXPODS   REPLICAS   AGE
keda-hpa-http-echo   Deployment/http-echo   <unknown>/10 (avg)   1         10        0          5m46s
```

2. Verify that the application has zero replicas:

```bash
kubectl get pod -n demo-app
No resources found in demo-app namespace.
```

3. Send a request to trigger scale-from-zero:

```bash
curl -v -H "Content-Type: application/json" -X GET -d '{"foo":"bar"}' https://http-echo-keda.${DOMAIN}/
```

The first request may take up to 30–60 seconds while the Pod starts. The response confirms the request was not lost:

```json
{
  "timestamp": "2026-03-30T13:12:32.252Z",
  "pod": "http-echo-abc123-xyz",
  "body": "{\"foo\":\"bar\"}",
  "message": "Request handled successfully by KEDA HTTP Add-on demo"
}
```

4. Observe the demo application scaling up from zero:

```bash
kubectl get pod -n demo-app -w
NAME                         READY   STATUS            RESTARTS   AGE
http-echo-677d479d69-sjd95   0/2     Pending           0          0s
http-echo-677d479d69-sjd95   0/2     Init:0/2          0          1s
http-echo-677d479d69-sjd95   0/2     PodInitializing   0          4s
http-echo-677d479d69-sjd95   1/2     Running           0          5s
http-echo-677d479d69-sjd95   2/2     Running           0          21s
```

```bash
kubectl get hpa -n demo-app -w
NAME                 REFERENCE              TARGETS              MINPODS   MAXPODS   REPLICAS   AGE
keda-hpa-http-echo   Deployment/http-echo   <unknown>/10 (avg)   1         10        0          6m3s
keda-hpa-http-echo   Deployment/http-echo   1/10 (avg)           1         10        1          6m30s
```

Eventually, if there is no traffic, no Pods should be running after a configurable cooldown period:

```bash
kubectl get pod -n demo-app
No resources found in demo-app namespace.
```

## Clean Up

```bash
kubectl delete namespace demo-app
kubectl annotate keda -n kyma-system default \
  keda.kyma-project.io/addon-enabled=false --overwrite
```

This disables the HTTP Add-on and removes all its resources from the `keda` namespace. Other workloads in the namespace are not affected.