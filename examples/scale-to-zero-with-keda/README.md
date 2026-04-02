# Scale to Zero with KEDA HTTP Add-on

## Overview

This example demonstrates how to use the [KEDA HTTP Add-on](https://github.com/kedacore/http-add-on) on a Kyma cluster to achieve HTTP-based scale-to-zero and scale-from-zero for workloads, without losing any requests.
It uses:

- [KEDA HTTP Add-on](https://github.com/kedacore/http-add-on) to intercept, queue, and count incoming HTTP requests — enabling scale-to-zero and scale-from-zero without lost requests,
- [KEDA](https://keda.sh/) to drive the workload's scaling based on request rate metrics provided by the HTTP Add-on,
- A demo application ([hashicorp/http-echo](https://hub.docker.com/r/hashicorp/http-echo)) that returns request-specific information (request headers, Pod name, timestamp) to verify that no requests are lost during scaling,
- Istio service mesh to provide mTLS encryption between all components and to expose the application via API Gateway,
- API Gateway (APIRule v2) to route external HTTPS traffic to the HTTP Add-on's Interceptor.

It realises the following scenario:

![KEDA scaling scenario](assets/scale-to-0.png "Scenario")

1. An `HTTP request` arrives at the **API Gateway**, which routes traffic to the **Interceptor's Service**.
2. The **Interceptor** receives the request, counts it, and queues it if the target workload has 0 replicas.
3. The **External Scaler** pulls request count metrics from the **Interceptor** via gRPC.
4. **KEDA** reads metrics from the **External Scaler** and scales the target **Deployment** accordingly (including to/from zero).
5. **KEDA** reconciles the **ScaledObject** (auto-created from HTTPScaledObject) to manage the scaling behavior.
6. The **KEDA-HTTP Operator** watches **HTTPScaledObject** resources and configures all add-on components (Interceptor routing, ScaledObject, External Scaler).
7. Once the **Deployment** has ready replicas, the **Interceptor** forwards the queued request to the application **Service**, which routes it to the running **Pod**.


## Prerequisites

- Kyma as the target Kubernetes runtime.
- [Keda, Istio and API Gateway modules installed](https://kyma-project.io/02-get-started/01-quick-install.html#steps)


## Procedure



1. Use the following command to add the Helm repository:
```bash
helm repo add kedacore https://kedacore.github.io/charts
helm repo update
```

2. Create a namespace with Istio sidecar injection enabled:
```bash
kubectl create namespace http-add-on
kubectl label namespace http-add-on istio-injection=enabled
```
Install the http-add-on:
```bash
helm install http-add-on kedacore/keda-add-ons-http \
  --namespace http-add-on
  ```

###  2. Configure Istio compatibility

  The add-on components use gRPC on port 9090 for internal communication. Istio sidecar intercepts this traffic and breaks gRPC health checks, causing `CrashLoopBackOff`. Exclude port 9090 from sidecar interception on all add-on deployments:
  - keda-add-ons-http-controller-manager
  - keda-add-ons-http-external-scaler
  - keda-add-ons-http-interceptor

  ```bash
  kubectl patch deployment <http-add-on-deployment> -n http-add-on \
  --type=merge \
  -p '{"spec":{"template":{"metadata":{"annotations":{"traffic.sidecar.istio.io/excludeInboundPorts":"9090"}}}}}'
  ```
 Wait until all pods are running the Istio sidecar (2/2):
 
 ```bash
 kubectl get pods -n http-add-on
NAME                                                   READY   STATUS    RESTARTS      AGE
keda-add-ons-http-controller-manager-87b4477f6-5bppt   2/2     Running   0             1h
keda-add-ons-http-external-scaler-5968cb8456-hr5ft     2/2     Running   0             1h
keda-add-ons-http-external-scaler-5968cb8456-p95bm     2/2     Running   0             1h
keda-add-ons-http-external-scaler-5968cb8456-zkmmn     2/2     Running   0             1h
keda-add-ons-http-interceptor-5ffc68f88d-fsv6m         2/2     Running   0             1h
keda-add-ons-http-interceptor-5ffc68f88d-gn96w         2/2     Running   0             1h
keda-add-ons-http-interceptor-5ffc68f88d-tp2zf         2/2     Running   0             1h
 ```

###  3. Deploy the example resources
Edit the `k8s-resources/apirule.yaml`, `k8s-resources/httpscaledobject.yaml` and `k8s-resources/envoyfilter.yaml` files to fill in the hosts value.

Apply the example resources from `./k8s-resources` directory:

```bash
kubectl apply -f ./k8s-resources
```
### 4. Key resources explained

| Resource | Kind | Description |
|---|---|---|
| `apirule.yaml` | `APIRule` | Routes external HTTPS traffic from `https://http-echo-keda.<YOUR_DOMAIN>` through Istio Ingress Gateway directly to the HTTP Add-on Interceptor's Service (`keda-add-ons-http-interceptor-proxy`) in the `http-add-on` namespace. The Interceptor then handles request queuing and forwarding to the application. |
| `httpscaledobject.yaml` | `HTTPScaledObject` | Configures the HTTP Add-on to scale the `http-echo` Deployment based on the incoming request rate. Sets `min: 0` to allow scale-to-zero and `max: 10` for scale-out. The operator automatically creates a corresponding KEDA `ScaledObject` for this resource. |
| `envoyfilter.yaml` | `EnvoyFilter` | Configures the Istio Ingress Gateway to automatically retry requests that fail with `5xx`, `connect-failure`, or `reset` — up to 100 times with a 3-second per-try timeout. This is critical for cold start: when the application is scaling from zero, the first forwarded request may arrive before the Pod is ready. The retry policy ensures the request is eventually delivered without returning an error to the client. |
| `demo-app.yaml` | `Deployment` + `Service` | Deploys a lightweight Node.js HTTP server that returns a JSON response with the handling Pod's name, a timestamp, and the request body. This allows you to verify that every request was processed and no request was lost during scale-from-zero. |

## Test the application 

Initially, the application Pod is scaled down to zero.

1. List HPA for the demo application and check that the current replica count is zero:

```bash
NAME                 REFERENCE              TARGETS              MINPODS   MAXPODS   REPLICAS   AGE
keda-hpa-http-echo   Deployment/http-echo   <unknown>/10 (avg)   1         10        0          5m46s
```

2. Verify that the application has zero replicas:

```bash
kubectl get pod -n  demo-app
No resources found in demo-app namespace.
```

3. Send a request to trigger scale-from-zero:

Call the HTTP proxy once:

```bash
curl -v -H "Content-Type: application/json" -X GET -d '{"foo":"bar"}' https://incoming.{your_cluster_domain}
```

The first request may take up to 30–60 seconds while the Pod starts. The response confirms the request was not lost:

```bash
{
  "timestamp": "2026-03-30T13:12:32.252Z",
  "pod": "http-echo-abc123-xyz",
  "body": "{\"foo\":\"bar\"}",
  "message": "Request handled successfully by KEDA HTTP Add-on demo"
}
```

4. Observe the demo application scaling up from zero. You can notice it by watching application Pods or HPA.

```bash 
kubectl get pod -n  demo-app -w
NAME                         READY   STATUS    RESTARTS   AGE
http-echo-677d479d69-sjd95   0/2     Pending   0          0s
http-echo-677d479d69-sjd95   0/2     Pending   0          0s
http-echo-677d479d69-sjd95   0/2     Init:0/2   0          0s
http-echo-677d479d69-sjd95   0/2     Init:0/2   0          1s
http-echo-677d479d69-sjd95   0/2     Init:1/2   0          2s
http-echo-677d479d69-sjd95   0/2     Init:1/2   0          3s
http-echo-677d479d69-sjd95   0/2     PodInitializing   0          4s
http-echo-677d479d69-sjd95   0/2     PodInitializing   0          4s
http-echo-677d479d69-sjd95   1/2     Running           0          5s
http-echo-677d479d69-sjd95   2/2     Running           0          21s
```

```bash hpa
kubectl get hpa -n demo-app -w
NAME                 REFERENCE              TARGETS              MINPODS   MAXPODS   REPLICAS   AGE
keda-hpa-http-echo   Deployment/http-echo   <unknown>/10 (avg)   1         10        0          6m3s
keda-hpa-http-echo   Deployment/http-echo   1/10 (avg)           1         10        1          6m30s
```


Eventually, if there is no traffic, no Pods should be running after a configurable cooldown period

```bash
kubectl get pod -n  demo-app
No resources found in demo-app namespace.
```

## Clean up

```bash
kubectl delete namespace demo-app
helm uninstall http-add-on -n http-add-on
kubectl delete namespace http-add-on
```