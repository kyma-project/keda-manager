# Dynatrace Request Rate Scaler Example

This example helps you set up the infrastructure needed for request rate-based autoscaling using Dynatrace, Keda and Istio in a Kyma environment.

## Prerequisites

- Kyma runtime with Keda, Istio and API Gateway enabled
- Dynatrace instance available

## Contents

`/k8s-resources/application.yaml` defines:
 - the `keda-dynatrace` namespace to isolate the example resources.
 - `httpbin` sample application with the Istio sidecar injected.The sidecar exposes Prometheus-compatible metrics on port 15090
 - `httpbin` service that exposes the application on port 80
 - `httpbin-metrics` service that exposes the Istio sidecar metrics on port 15090. Service includes Dynatrace annotations to enable metric scraping from the Istio sidecar.
 - `httpbin` APIRule to expose the `httpbin` service externally via the Kyma API Gateway.

`/k8s-resources/scaler.yaml` defines:
 - Secret with `host` and `token` keys (token must have `metrics.read` permission)
 - `TriggerAuthentication` that references the secret
 - `ScaledObject` that defines target workload, references `TriggerAuthentication` object and configures the [Dynatrace trigger](https://keda.sh/docs/2.15/scalers/dynatrace/)

## How It Works

1. **Metric Exposure**  
   The Istio sidecar in the `httpbin` deployment exposes metrics on port 15090 at the `/stats/prometheus` endpoint.  
   The Service is annotated so Dynatrace can discover and scrape these metrics.

2. **Dynatrace Integration**  
   Dynatrace will scrape the Istio metrics, including request rate, from the annotated Service.

3. **Keda Scaling**  
   Keda `ScaledObject` with a `dynatrace` trigger autoscales the `httpbin` deployment based on incoming request rate.

## Usage

1. Apply the manifests in this folder to your Kubernetes cluster:

```bash
   kubectl apply -f ./k8s-resource
```

2. Generate http trafic of 2 requests per second by running:

```bash
   export CLUSTER_DOMAIN="<your cluster domain>"
   while true; do sleep 0.5; wget -q -O- "https://httpbin.${CLUSTER_DOMAIN}/get" ; done
```

3. Observe the metric value in dynatrace UI:

![Dynatrace Metric Visualization](assets/dynatrace-ui.png)

The measured value divided by the threshold value should give the number of desired replicas.
`150 / 50 = 3`.

4. Observe how the httpbin pods get autoscaled to 3 replicas:

```bash
kubectl get pods -n keda-dynatrace -w
NAME                       READY   STATUS    RESTARTS   AGE
httpbin-5fb66474c4-4v2xj   2/2     Running   0          65s
httpbin-5fb66474c4-llggp   2/2     Running   0          48m
httpbin-5fb66474c4-zdqg2   2/2     Running   0          64s
```