# Scaling HTTP Workloads to Zero With the KEDA HTTP Add-on

## Prerequisites

- A Kyma runtime cluster.
- [Keda, Istio, and API Gateway modules installed](https://kyma-project.io/02-get-started/01-quick-install.html#steps).
- The `kubectl` and `curl` CLIs installed locally.

## Context

The [KEDA HTTP Add-on](https://github.com/kedacore/http-add-on) extends KEDA with HTTP-based scale-to-zero and scale-from-zero for workloads, without losing requests. Its **Interceptor** queues incoming requests, the **External Scaler** exposes the request rate to KEDA over gRPC, and the **Operator** reconciles `HTTPScaledObject` resources to drive the application's scaling. Combined with Istio service mesh and API Gateway, this example exposes a demo application over HTTPS, drives it to zero replicas during idle periods, and scales it back up on the first incoming request.

This example installs the following resources from the `k8s-resources` directory:

| File | Kind | Purpose |
|---|---|---|
| `apirule.yaml` | `APIRule` | Routes external HTTPS traffic from `https://http-echo-keda.<YOUR_DOMAIN>` through the Istio Ingress Gateway to the HTTP Add-on Interceptor's Service (`keda-add-ons-http-interceptor-proxy`). |
| `httpscaledobject.yaml` | `HTTPScaledObject` | Configures the HTTP Add-on to scale the `http-echo` Deployment by incoming request rate. `min: 0` enables scale-to-zero; `max: 10` caps scale-out. The Operator creates the corresponding KEDA `ScaledObject` automatically. |
| `envoyfilter.yaml` | `EnvoyFilter` | Configures the Istio Ingress Gateway to retry requests that fail with `5xx`, `connect-failure`, or `reset` — up to 100 times with a 3-second per-try timeout. This covers the cold-start window when the first forwarded request may arrive before the Pod is ready. |
| `demo-app.yaml` | `Deployment` + `Service` | Deploys a lightweight Node.js HTTP server that returns a JSON response with the handling Pod's name, a timestamp, and the request body. Use this to verify that no request is lost during scale-from-zero. |

## Procedure

1. Enable the HTTP Add-on by annotating the Keda custom resource (CR). The Keda Manager installs the add-on in the specified namespace:

   ```bash
   kubectl annotate keda -n kyma-system default \
     keda.kyma-project.io/addon-enabled=true 
   ```

2. Enable Istio sidecar injection on the HTTP Add-on Deployments. The demo application runs in an Istio-injected namespace and accepts only mTLS traffic, so the Interceptor and Scaler must also have sidecars to reach it:

   ```bash
   kubectl annotate keda -n kyma-system default \
     keda.kyma-project.io/addon-istio-injection=true
   ```

3. Verify that the add-on condition is `True`:

   ```bash
   kubectl get keda -n kyma-system default -o jsonpath='{.status.conditions[?(@.type=="Addon")].status}'
   ```

   4. Verify that all add-on Pods are running with the Istio sidecar (`2/2`). Filter by the `kyma-project.io/module=keda` label that the Keda Manager stamps on every add-on resource:

   ```bash
   kubectl get pods -n kyma-system -l kyma-project.io/module=keda
   ```

5. Export your cluster domain:

   ```bash
   export DOMAIN=$(kubectl get configmap shoot-info -n kube-system -o jsonpath='{.data.domain}')
   echo $DOMAIN
   ```

6. In `k8s-resources/httpscaledobject.yaml` and `k8s-resources/envoyfilter.yaml`, replace `<YOUR_DOMAIN>` with your cluster domain. In `envoyfilter.yaml`, the vhost name must use the format `http-echo-keda.<YOUR_DOMAIN>:443`.


7. Create the `demo-app` namespace and enable Istio sidecar injection on it:

   ```bash
   kubectl create namespace demo-app
   kubectl label namespace demo-app istio-injection=enabled
   ```

7. Apply the example resources:

   ```bash
   kubectl apply -f ./k8s-resources
   ```

9. Wait until the application Pod scales to zero, then send a request to trigger scale-from-zero:

   ```bash
   curl -v -H "Content-Type: application/json" -X GET -d '{"foo":"bar"}' https://http-echo-keda.${DOMAIN}/
   ```

   The first request can take 30–60 seconds while the Pod starts.

## Result

The `curl` command returns a JSON response that confirms the request reached a freshly scaled-up Pod:

```json
{
  "timestamp": "2026-03-30T13:12:32.252Z",
  "pod": "http-echo-abc123-xyz",
  "body": "{\"foo\":\"bar\"}",
  "message": "Request handled successfully by KEDA HTTP Add-on demo"
}
```

While the request is being served, `http-echo` scales from zero to one replica:

```bash
kubectl get hpa -n demo-app -w
NAME                 REFERENCE              TARGETS              MINPODS   MAXPODS   REPLICAS   AGE
keda-hpa-http-echo   Deployment/http-echo   <unknown>/10 (avg)   1         10        0          6m3s
keda-hpa-http-echo   Deployment/http-echo   1/10 (avg)           1         10        1          6m30s
```

After traffic stops and the configurable cooldown period elapses, the application scales back to zero:

```bash
kubectl get pod -n demo-app
No resources found in demo-app namespace.
```

## Cleanup

Remove the demo namespace and disable the HTTP Add-on:

```bash
kubectl delete namespace demo-app
kubectl annotate keda -n kyma-system default \
  keda.kyma-project.io/addon-enabled=false --overwrite
```

This removes all add-on resources from the cluster. Other workloads in the add-on namespace are not affected.

## Related Information

- [KEDA HTTP Add-on](../../docs/user/07-10-http-add-on.md)
- [HTTP Add-on Returns 503 Errors During Cold Start](../../docs/user/troubleshooting-guides/07-10-cold-start-503-errors.md)
- [KEDA HTTP Add-on on GitHub](https://github.com/kedacore/http-add-on)
