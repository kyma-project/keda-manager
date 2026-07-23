# Scale Applications Using SAP BTP Cloud Logging Service Metrics

This tutorial shows how to configure KEDA to autoscale a Kubernetes workload using application metrics stored in SAP BTP Cloud Logging Service (CLS) as the scaling signal.

SAP BTP Cloud Logging Service is a managed observability backend built on OpenSearch. When your application metrics already flow into CLS through the Kyma Telemetry module, you can use those metrics as autoscaling signals without running a separate metrics store.

KEDA 2.20 ships a native `opensearch` scaler that queries CLS directly using an inline query. This tutorial walks you through the full setup: from a demo app that emits a `queue_depth` metric, through Telemetry scraping, to a `ScaledObject` that scales the workload based on that metric.

## Prerequisites

- The Keda, Telemetry, and BTP Operator modules are enabled in your Kyma cluster. See [Enable and Disable a Kyma Module](https://help.sap.com/docs/btp/sap-business-technology-platform/enable-and-disable-kyma-module?locale=en-US).
- Your subaccount has an entitlement for Cloud Logging Service with the `standard` plan. See [Configure Entitlements and Quotas for Subaccounts](https://help.sap.com/docs/btp/sap-business-technology-platform/configure-entitlements-and-quotas-for-subaccounts).
- `kubectl` is installed and configured to access your Kyma cluster.

## Steps

### Create a CLS Instance and Credentials

Choose the setup that matches your deployment strategy:

- **Option A — BTP Cockpit:** Create the CLS instance in a separate subaccount using BTP Cockpit. The instance and its credentials are independent of any Kyma cluster, so they survive cluster replacement or deletion.
- **Option B — BTP Operator:** Create the CLS instance directly in your Kyma cluster using the BTP Operator. Contact your Cloud Provider to confirm that the CLS backend API is available in your region before using this option.

#### Option A: Create CLS via BTP Cockpit

1. In SAP BTP Cockpit, go to **Services** → **Instances and Subscriptions** and choose **Create**.

2. Configure the service instance and choose **Next**:
   - **Service**: Cloud Logging
   - **Plan**: standard
   - **Runtime Environment**: Other
   - **Instance Name**: Enter a name, for example `cloud-logging`

3. In the **Parameters** field, enter the following JSON and choose **Create**:

    ```json
    {
        "backend": {
            "api_enabled": true,
            "max_data_nodes": 2
        },
        "ingest_otlp": {
            "enabled": true
        }
    }
    ```

4. Wait until the instance status changes to **Created**.

5. In the instance row, choose the **...** (Actions) menu and select **Create Service Binding**. Enter a name for the binding, for example `cloud-logging-binding`, and choose **Create**.

6. Once the binding is created, choose **View Credentials** from the same **...** menu. Note the following values:

   | Key | Description |
   |---|---|
   | `backend-endpoint` | OpenSearch REST API endpoint |
   | `backend-username` | Username for OpenSearch REST API authentication |
   | `backend-password` | Password for OpenSearch REST API authentication |
   | `ingest-otlp-endpoint` | OTLP ingest endpoint for the Telemetry module |
   | `ingest-otlp-cert` | Client certificate for mTLS |
   | `ingest-otlp-key` | Client key for mTLS |

7. Create a namespace and a Kubernetes Secret with the credentials:

    ```bash
    kubectl create namespace cls
    ```

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: v1
    kind: Secret
    metadata:
      name: cloud-logging-binding
      namespace: cls
    type: Opaque
    stringData:
      backend-endpoint: "<BACKEND_ENDPOINT>"
      backend-username: "<BACKEND_USERNAME>"
      backend-password: "<BACKEND_PASSWORD>"
      ingest-otlp-endpoint: "<INGEST_OTLP_ENDPOINT>"
      ingest-otlp-cert: |
        <INGEST_OTLP_CERT>
      ingest-otlp-key: |
        <INGEST_OTLP_KEY>
    EOF
    ```

    Replace each placeholder with the corresponding value from BTP Cockpit. For `ingest-otlp-cert` and `ingest-otlp-key`, paste the full PEM content including the `-----BEGIN ...-----` and `-----END ...-----` lines.

#### Option B: Create CLS via BTP Operator

> [!NOTE]
> This option requires the CLS backend API endpoint to be available in your region. Contact your Cloud Provider to confirm availability before proceeding.

1. Create a namespace for CLS resources and a `ServiceInstance` for Cloud Logging:

    ```bash
    kubectl create namespace cls
    ```

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceInstance
    metadata:
      name: cloud-logging
      namespace: cls
    spec:
      serviceOfferingName: cloud-logging
      servicePlanName: standard
      parameters:
        backend:
          api_enabled: true
          max_data_nodes: 2
        ingest_otlp:
          enabled: true
    EOF
    ```

2. Wait until the instance is ready:

    ```bash
    kubectl get serviceinstance cloud-logging -n cls -w
    ```

    You should get a result similar to this example:

    ```bash
    NAME             OFFERING        PLAN       STATUS    AGE
    cloud-logging    cloud-logging   standard   Ready    2m
    ```

3. Create a `ServiceBinding` to generate the credentials secret:

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: services.cloud.sap.com/v1
    kind: ServiceBinding
    metadata:
      name: cloud-logging-binding
      namespace: cls
    spec:
      serviceInstanceName: cloud-logging
    EOF
    ```

4. Verify that the binding secret was created and contains the OTLP keys:

    ```bash
    kubectl get secret cloud-logging-binding -n cls -o jsonpath='{.data}' | jq 'keys'
    ```

    The secret must contain `backend-endpoint`, `backend-username`, `backend-password`, `ingest-otlp-endpoint`, `ingest-otlp-cert`, and `ingest-otlp-key`.

### Deploy the Demo Application

The demo application exposes a Prometheus-format `queue_depth` gauge metric at the `/metrics` endpoint. KEDA uses this value — as stored in CLS — to determine the desired replica count.

The `QUEUE_DEPTH` value is set by an init container at pod startup. To change the value, update the env var and restart the Pod.

1. Save the demo application manifest to a file and apply it:

    ```bash
    cat > /tmp/demo-app.yaml << 'EOF'
    apiVersion: v1
    kind: Namespace
    metadata:
      name: keda-cls-demo
    ---
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: fake-metrics-nginx-config
      namespace: keda-cls-demo
    data:
      default.conf.template: |
        server {
            listen 8080;
            root /usr/share/nginx/html;

            location /metrics {
                default_type "text/plain; version=0.0.4; charset=utf-8";
                try_files /metrics.txt =404;
            }

            location /health {
                default_type text/plain;
                return 200 "OK";
            }

            location / {
                return 404;
            }
        }
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: fake-metrics
      namespace: keda-cls-demo
      labels:
        app: fake-metrics
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: fake-metrics
      template:
        metadata:
          labels:
            app: fake-metrics
        spec:
          initContainers:
          - name: generate-metrics
            image: busybox:1.36
            command:
            - sh
            - -c
            - printf '# HELP queue_depth The current depth of the queue\n# TYPE queue_depth gauge\nqueue_depth %d\n' "$QUEUE_DEPTH" > /data/metrics.txt
            env:
            - name: QUEUE_DEPTH
              value: "10"
            volumeMounts:
            - name: metrics-data
              mountPath: /data
          containers:
          - name: fake-metrics
            image: nginx:alpine
            ports:
            - containerPort: 8080
            resources:
              requests:
                memory: "64Mi"
                cpu: "100m"
              limits:
                memory: "128Mi"
                cpu: "200m"
            volumeMounts:
            - name: nginx-config
              mountPath: /etc/nginx/templates
            - name: metrics-data
              mountPath: /usr/share/nginx/html
            livenessProbe:
              httpGet:
                path: /health
                port: 8080
              initialDelaySeconds: 5
              periodSeconds: 10
            readinessProbe:
              httpGet:
                path: /health
                port: 8080
              initialDelaySeconds: 5
              periodSeconds: 5
          volumes:
          - name: nginx-config
            configMap:
              name: fake-metrics-nginx-config
          - name: metrics-data
            emptyDir: {}
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: fake-metrics
      namespace: keda-cls-demo
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      selector:
        app: fake-metrics
      ports:
      - port: 8080
        protocol: TCP
        targetPort: 8080
      type: ClusterIP
    EOF

    kubectl apply -f /tmp/demo-app.yaml
    ```

2. Verify that the Pod is running:

    ```bash
    kubectl get pods -n keda-cls-demo
    ```

    You should get a result similar to this example:

    ```bash
    NAME                           READY   STATUS    RESTARTS   AGE
    fake-metrics-<hash>            1/1     Running   0          30s
    ```

3. Confirm the `/metrics` endpoint is reachable:

    ```bash
    kubectl run curl-test --image=curlimages/curl --rm -it --restart=Never --quiet \
      -- curl -s http://fake-metrics.keda-cls-demo.svc.cluster.local:8080/metrics
    ```

    You should get a result similar to this example:

    ```bash
    # HELP queue_depth The current depth of the queue
    # TYPE queue_depth gauge
    queue_depth 10
    ```

### Configure the Telemetry Module to Forward Metrics to CLS

The Kyma Telemetry module scrapes Prometheus metrics from annotated Services and forwards them to your CLS instance. The Prometheus scraping annotations are already included in the demo application manifest.

Create a `MetricPipeline` resource that sends the scraped metrics to CLS using the OTLP credentials from the binding secret:

```bash
kubectl apply -f - <<EOF
apiVersion: telemetry.kyma-project.io/v1beta1
kind: MetricPipeline
metadata:
  name: cls-metric-pipeline
spec:
  input:
    prometheus:
      enabled: true
      namespaces:
        include:
          - keda-cls-demo
    istio:
      enabled: false
    runtime:
      enabled: false
    otlp:
      enabled: true
  output:
    otlp:
      endpoint:
        valueFrom:
          secretKeyRef:
            name: cloud-logging-binding
            namespace: cls
            key: ingest-otlp-endpoint
      tls:
        cert:
          valueFrom:
            secretKeyRef:
              name: cloud-logging-binding
              namespace: cls
              key: ingest-otlp-cert
        key:
          valueFrom:
            secretKeyRef:
              name: cloud-logging-binding
              namespace: cls
              key: ingest-otlp-key
EOF
```

Verify the pipeline is ready:

```bash
kubectl get metricpipeline cls-metric-pipeline
```

You should get a result similar to this example:

```bash
NAME                  CONFIGURATION GENERATED   GATEWAY HEALTHY   AGENT HEALTHY   FLOW HEALTHY   AGE
cls-metric-pipeline   True                      True              True            True           2m
```

In the CLS OpenSearch Dashboards, confirm that the `queue_depth` metric is arriving:

1. Get the Dashboards URL and credentials:
   - **Option A:** Find the `dashboards-endpoint`, `dashboards-username`, and `dashboards-password` values in BTP Cockpit under **View Credentials** for your CLS instance.
   - **Option B:** Extract them from the binding secret:

    ```bash
    echo "URL: https://$(kubectl get secret cloud-logging-binding -n cls -o jsonpath='{.data.dashboards-endpoint}' | base64 -d)"
    echo "Username: $(kubectl get secret cloud-logging-binding -n cls -o jsonpath='{.data.dashboards-username}' | base64 -d)"
    echo "Password: $(kubectl get secret cloud-logging-binding -n cls -o jsonpath='{.data.dashboards-password}' | base64 -d)"
    ```

2. Open the URL in your browser and log in with the credentials.

3. In the navigation menu, go to **Discover**, select the `metrics-otel-v1-*` index pattern, and filter for documents with `name: queue_depth`. The metric should appear within 1–2 minutes after the MetricPipeline becomes healthy.

### Create a KEDA TriggerAuthentication for CLS

KEDA must authenticate with the CLS OpenSearch REST API to run queries. Store the CLS credentials in a Kubernetes `Secret` and reference them from a `TriggerAuthentication`.

The credentials are available in the CLS service binding secret. The following keys are used:

| Key | Description |
|---|---|
| `backend-endpoint` | OpenSearch REST API endpoint |
| `backend-username` | Username for OpenSearch REST API authentication |
| `backend-password` | Password for OpenSearch REST API authentication |

1. Create a `Secret` with your CLS OpenSearch credentials:

    ```bash
    kubectl create secret generic cls-keda-auth \
      --from-literal=username=$(kubectl get secret cloud-logging-binding -n cls -o jsonpath='{.data.backend-username}' | base64 -d) \
      --from-literal=password=$(kubectl get secret cloud-logging-binding -n cls -o jsonpath='{.data.backend-password}' | base64 -d) \
      -n keda-cls-demo
    ```

2. Create a `TriggerAuthentication` that references the secret:

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: keda.sh/v1alpha1
    kind: TriggerAuthentication
    metadata:
      name: cls-trigger-auth
      namespace: keda-cls-demo
    spec:
      secretTargetRef:
        - parameter: username
          name: cls-keda-auth
          key: username
        - parameter: password
          name: cls-keda-auth
          key: password
    EOF
    ```

### Create the ScaledObject

The `ScaledObject` tells KEDA to query CLS for the latest `queue_depth` value and scale the demo application accordingly.

1. Export the OpenSearch endpoint from your CLS service binding secret:

    ```bash
    export CLS_OPENSEARCH_ENDPOINT=https://$(kubectl get secret cloud-logging-binding -n cls -o jsonpath='{.data.backend-endpoint}' | base64 -d)
    export CLS_OPENSEARCH_USERNAME=$(kubectl get secret cloud-logging-binding -n cls -o jsonpath='{.data.backend-username}' | base64 -d)
    ```

2. Create the `ScaledObject`:

    ```bash
    cat > /tmp/scaled-object.yaml << EOF
    apiVersion: keda.sh/v1alpha1
    kind: ScaledObject
    metadata:
      name: cls-queue-depth-scaler
      namespace: keda-cls-demo
    spec:
      scaleTargetRef:
        name: fake-metrics
      minReplicaCount: 1
      maxReplicaCount: 10
      triggers:
        - type: opensearch
          metadata:
            addresses: "${CLS_OPENSEARCH_ENDPOINT}"
            username: "${CLS_OPENSEARCH_USERNAME}"
            index: "metrics-otel-v1-*"
            query: |
              {
                "size": 0,
                "query": {
                  "bool": {
                    "filter": [
                      { "term": { "name": "queue_depth" } },
                      { "range": { "time": { "gte": "now-1m" } } }
                    ]
                  }
                },
                "aggs": {
                  "latest_value": {
                    "max": { "field": "value" }
                  }
                }
              }
            valueLocation: "aggregations.latest_value.value"
            targetValue: "10"
          authenticationRef:
            name: cls-trigger-auth
    EOF

    kubectl apply -f /tmp/scaled-object.yaml
    ```

    > [!NOTE]
    > The `targetValue` of `10` means KEDA targets one replica per 10 units of `queue_depth`. With a `queue_depth` of 42, KEDA targets 5 replicas (`ceil(42/10)`).

3. Verify that KEDA has picked up the scaler:

    ```bash
    kubectl get scaledobject cls-queue-depth-scaler -n keda-cls-demo
    ```

    You should get a result similar to this example:

    ```bash
    NAME                     SCALETARGETKIND      SCALETARGETNAME   MIN   MAX   TRIGGERS     READY   ACTIVE
    cls-queue-depth-scaler   apps/v1.Deployment   fake-metrics      1     10    opensearch   True    True
    ```

### Observe Autoscaling in Action

1. Check the KEDA-managed HPA:

    ```bash
    kubectl get hpa -n keda-cls-demo
    ```

    You should get a result similar to this example:

    ```bash
    NAME                              REFERENCE                    TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
    keda-hpa-cls-queue-depth-scaler   Deployment/fake-metrics      10/10     1         10        1          2m
    ```

2. Simulate a metric spike by updating the `QUEUE_DEPTH` environment variable and restarting the Pod:

    ```bash
    kubectl set env deployment/fake-metrics QUEUE_DEPTH=80 -n keda-cls-demo
    kubectl rollout restart deployment/fake-metrics -n keda-cls-demo
    ```

    After the next Telemetry scrape and CLS ingestion cycle (typically within 1–2 minutes), KEDA queries CLS and adjusts the replica count.

3. Watch the Pods scale up:

    ```bash
    kubectl get pods -n keda-cls-demo -w
    ```

4. Set the metric back to a lower value to observe scale-down:

    ```bash
    kubectl set env deployment/fake-metrics QUEUE_DEPTH=5 -n keda-cls-demo
    kubectl rollout restart deployment/fake-metrics -n keda-cls-demo
    ```

    After the cooldown period, the replica count returns to the minimum of 1.

## Result

Your workload scales automatically in response to the `queue_depth` metric stored in SAP BTP Cloud Logging Service. KEDA polls CLS on each reconciliation cycle and adjusts the HPA target accordingly.

## Clean Up

Remove all resources created during this tutorial:

```bash
kubectl delete namespace keda-cls-demo
kubectl delete metricpipeline cls-metric-pipeline
kubectl delete namespace cls
```

If you used Option B, also delete the service binding and instance:

```bash
kubectl delete servicebinding cloud-logging-binding -n cls
kubectl delete serviceinstance cloud-logging -n cls
```

If you used Option A, delete the CLS instance in BTP Cockpit under **Services** → **Instances and Subscriptions**.
