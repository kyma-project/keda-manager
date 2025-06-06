# Dynatrace Request Rate Scaler Example

This example helps you set up the infrastructure needed for request rate-based autoscaling using Dynatrace, Keda and Istio in a Kyma environment.

## Prerequisites

- Kyma runtime with Keda, Istio and API Gateway enabled
- Dynatrace instance available

## Contents

`/k8s-resources/application.yaml` defines:
 - the `keda-dynatrace` namespace to isolate the example resources.
 - `httpbin` sample application with the Istio sidecar injected.The sidecar exposes Prometheus-compatible metrics on port 15090
 - `httpbin` service that exposes the application on port 80 and the Istio sidecar metrics on port 15090. Service includes Dynatrace annotations to enable metric scraping from the Istio sidecar.
 - `httpbin` APIRule to expose the `httpbin` service externally via the Kyma API Gateway.

`/k8s-resources/scaler.yaml` defines:
 - Secret with `host` and `token` keys (token must have `metrics.read` permission)
 - TriggerAuthentication that references the secret
 - ScaledObject that defines target workload, references TriggerAuthentication object and configures the Dynatrace trigger

`/k8s-resources/request-generator.yaml` defines
 - a simple busybox based deployment to generate some traffic for testing

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

2. Observe how the httpbin pods get autoscaled:

```bash
kubectl get pods -n keda-dynatrace -w
NAME                                READY   STATUS    RESTARTS   AGE
httpbin-5fb66474c4-4zvlj            2/2     Running   0          57s
httpbin-5fb66474c4-5ldj2            2/2     Running   0          63m
httpbin-5fb66474c4-fvpm9            2/2     Running   0          87s
httpbin-5fb66474c4-rvcq7            2/2     Running   0          86s
httpbin-5fb66474c4-z27vw            2/2     Running   0          86s
request-generator-54b5656b5-fbhjv   2/2     Running   0          77s
```