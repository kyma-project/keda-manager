# Scale Applications Using SAP BTP Cloud Logging Service Metrics

This tutorial shows how to configure KEDA to autoscale a Kubernetes workload using application metrics stored in SAP BTP Cloud Logging Service (CLS) as the scaling signal.

SAP BTP Cloud Logging Service is a managed observability backend built on OpenSearch. When your application metrics already flow into CLS through the Kyma Telemetry module, you can use those metrics as autoscaling signals without running a separate metrics store.

KEDA's Elasticsearch scaler is compatible with OpenSearch and can query CLS directly. This tutorial walks you through the full setup: from a demo app that emits a `queue_depth` metric, through Telemetry scraping, to a `ScaledObject` that scales the workload based on that metric.

## Prerequisites

- The Keda and Telemetry modules are enabled in your Kyma cluster. See [Enable and Disable a Kyma Module](https://help.sap.com/docs/btp/sap-business-technology-platform/enable-and-disable-kyma-module?locale=en-US).
- An SAP BTP Cloud Logging Service instance is connected to your Kyma cluster. See [Integrate with SAP Cloud Logging](https://help.sap.com/docs/btp/sap-business-technology-platform/integrate-with-sap-cloud-logging).
- `kubectl` is installed and configured to access your Kyma cluster.

## Steps

### Deploy the Demo Application

The demo application exposes a Prometheus-format `queue_depth` gauge metric. KEDA uses this value — as stored in CLS — to determine the desired replica count.

1. Create a namespace for the demo:

    ```bash
    kubectl create namespace keda-cls-demo
    ```

2. Apply the demo application resources:

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/kyma-project/keda-manager/main/examples/cls-scaler/k8s-resources/demo-app.yaml -n keda-cls-demo
    ```

3. Verify that the Pod is running:

    ```bash
    kubectl get pods -n keda-cls-demo
    ```

    You should get a result similar to this example:

    ```bash
    NAME                        READY   STATUS    RESTARTS   AGE
    demo-app-<hash>             1/1     Running   0          30s
    ```

4. Confirm the `/metrics` endpoint is reachable:

    ```bash
    kubectl port-forward -n keda-cls-demo deploy/demo-app 8080:8080 &
    curl http://localhost:8080/metrics | grep queue_depth
    ```

    You should get a result similar to this example:

    ```bash
    # HELP queue_depth Simulated queue depth metric
    # TYPE queue_depth gauge
    queue_depth 42
    ```

### Configure the Telemetry Module to Scrape and Forward Metrics

The Kyma Telemetry module collects Prometheus metrics from annotated Pods and forwards them to your CLS instance.

1. Annotate the demo application's Pod template so that the Telemetry module discovers it:

    ```bash
    kubectl patch deployment demo-app -n keda-cls-demo --type=merge -p '{
      "spec": {
        "template": {
          "metadata": {
            "annotations": {
              "prometheus.io/scrape": "true",
              "prometheus.io/port": "8080",
              "prometheus.io/path": "/metrics"
            }
          }
        }
      }
    }'
    ```

2. Create a `MetricPipeline` resource that sends the scraped metrics to your CLS instance. Replace `<CLS-ENDPOINT>`, `<CLS-SECRET-NAME>`, and the secret key names with values from your CLS service binding:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: telemetry.kyma-project.io/v1alpha1
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
      output:
        otlp:
          endpoint:
            value: <CLS-ENDPOINT>
          auth:
            basic:
              user:
                valueFrom:
                  secretKeyRef:
                    name: <CLS-SECRET-NAME>
                    key: username
              password:
                valueFrom:
                  secretKeyRef:
                    name: <CLS-SECRET-NAME>
                    key: password
    EOF
    ```

3. Verify the pipeline is ready:

    ```bash
    kubectl get metricpipeline cls-metric-pipeline
    ```

    You should get a result similar to this example:

    ```bash
    NAME                   HEALTHY
    cls-metric-pipeline    True
    ```

4. In the CLS OpenSearch Dashboards, confirm that the `queue_depth` metric is arriving. Use the **Discover** view and filter for documents with a `metric.name` of `queue_depth`.

### Create a KEDA TriggerAuthentication for CLS

KEDA must authenticate with CLS to run OpenSearch queries. Store the CLS credentials in a Kubernetes `Secret` and reference them from a `TriggerAuthentication`.

1. Create a `Secret` with your CLS credentials. Replace the placeholder values with the actual credentials from your CLS service binding:

    ```bash
    kubectl create secret generic cls-keda-auth \
      --from-literal=username=<CLS-USERNAME> \
      --from-literal=password=<CLS-PASSWORD> \
      -n keda-cls-demo
    ```

2. Create a `TriggerAuthentication` that references the secret:

    ```bash
    cat <<EOF | kubectl apply -f -
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

The `ScaledObject` tells KEDA to query CLS for the `queue_depth` metric and scale the demo application accordingly.

1. Register a stored search template in CLS. Run the following request using the OpenSearch REST API or Dashboards Dev Tools, replacing `<CLS-OPENSEARCH-ENDPOINT>`, `<CLS-USERNAME>`, and `<CLS-PASSWORD>` with your CLS credentials:

    ```bash
    curl -X PUT "<CLS-OPENSEARCH-ENDPOINT>/_scripts/queue-depth-query" \
      -u "<CLS-USERNAME>:<CLS-PASSWORD>" \
      -H "Content-Type: application/json" -d '{
        "script": {
          "lang": "mustache",
          "source": {
            "query": {
              "bool": {
                "filter": [
                  { "term": { "metric.name": "queue_depth" } },
                  { "range": { "@timestamp": { "gte": "now-1m" } } }
                ]
              }
            },
            "aggs": {
              "latest_value": {
                "max": { "field": "metric.value" }
              }
            }
          }
        }
      }'
    ```

2. Create the `ScaledObject`. Replace `<CLS-OPENSEARCH-ENDPOINT>` with the OpenSearch REST endpoint of your CLS instance:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: keda.sh/v1alpha1
    kind: ScaledObject
    metadata:
      name: cls-queue-depth-scaler
      namespace: keda-cls-demo
    spec:
      scaleTargetRef:
        name: demo-app
      minReplicaCount: 1
      maxReplicaCount: 10
      triggers:
        - type: elasticsearch
          metadata:
            addresses: "<CLS-OPENSEARCH-ENDPOINT>"
            index: "metrics-*"
            searchTemplateName: "queue-depth-query"
            valueLocation: "hits.total.value"
            targetValue: "10"
          authenticationRef:
            name: cls-trigger-auth
    EOF
    ```

    > [!NOTE]
    > KEDA's `elasticsearch` scaler is compatible with OpenSearch. The `targetValue` of `10` means KEDA targets one replica per 10 units of `queue_depth`. With a `queue_depth` of 42, KEDA targets 5 replicas (`ceil(42/10)`).

3. Verify that KEDA has picked up the scaler:

    ```bash
    kubectl get scaledobject cls-queue-depth-scaler -n keda-cls-demo
    ```

    You should get a result similar to this example:

    ```bash
    NAME                     SCALETARGETKIND      SCALETARGETNAME   MIN   MAX   TRIGGERS       READY   ACTIVE
    cls-queue-depth-scaler   apps/v1.Deployment   demo-app          1     10    elasticsearch  True    True
    ```

### Observe Autoscaling in Action

1. Check the KEDA-managed HPA:

    ```bash
    kubectl get hpa -n keda-cls-demo
    ```

    You should get a result similar to this example:

    ```bash
    NAME                              REFERENCE             TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
    keda-hpa-cls-queue-depth-scaler   Deployment/demo-app   42/10     1         10        5          2m
    ```

2. Simulate a metric spike by updating the `QUEUE_DEPTH` environment variable in the demo app:

    ```bash
    kubectl set env deployment/demo-app QUEUE_DEPTH=80 -n keda-cls-demo
    ```

    After the next Telemetry scrape and CLS ingestion cycle (typically within 1–2 minutes), KEDA queries CLS and adjusts the replica count.

3. Watch the Pods scale up:

    ```bash
    kubectl get pods -n keda-cls-demo -w
    ```

4. Set the metric back to a lower value to observe scale-down:

    ```bash
    kubectl set env deployment/demo-app QUEUE_DEPTH=5 -n keda-cls-demo
    ```

    After the cooldown period, the replica count returns to the minimum of 1.

## Result

Your workload scales automatically in response to the `queue_depth` metric stored in SAP BTP Cloud Logging Service. KEDA polls CLS on each reconciliation cycle and adjusts the HPA target accordingly.

## Clean Up

Remove all resources created during this tutorial:

```bash
kubectl delete namespace keda-cls-demo
kubectl delete metricpipeline cls-metric-pipeline
```
