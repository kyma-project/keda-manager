## Scenario

This example demonstrates event-driven autoscaling of a consumer deployment with the KEDA
`nats-jetstream` scaler.

- A producer publishes messages to the `orders.created` subject in batches.
- A consumer processes messages from a JetStream pull consumer.
- KEDA scales the consumer deployment based on JetStream consumer lag.

## Prerequisites

- `KUBECONFIG` environment variable pointing to a Kubernetes cluster.
- Keda and NATS modules installed in the Kyma cluster.
   - For unmanaged Kyma, see [Kyma Quick Install][kyma-quick-install].
   - For managed Kyma, see [Enable and Disable Kyma Module][kyma-managed-modules].

[kyma-quick-install]: https://kyma-project.io/02-get-started/01-quick-install.html
[kyma-managed-modules]: https://help.sap.com/docs/btp/sap-business-technology-platform/enable-and-disable-kyma-module?locale=en-US

## How the Sample App Works

This sample deploys four main elements in the `nats-jetstream-demo` namespace:

- `setup-jetstream` Job: creates JetStream stream `demo-stream` and pull consumer `demo-consumer`.
- `jetstream-producer` Deployment: publishes messages to subject `orders.created` in batches.
- `jetstream-consumer` Deployment: pulls and acknowledges messages from JetStream.
- `jetstream-lag-scaler` ScaledObject: uses the KEDA `nats-jetstream` trigger and scales `jetstream-consumer`.

The runtime flow is:

1. Producer sends messages into NATS JetStream (`orders.created` -> `demo-stream`).
2. If consumer replicas are low (or zero), the message backlog (lag) grows.
3. KEDA reads lag metrics from the NATS monitoring endpoint.
4. KEDA updates HPA desired replicas for `jetstream-consumer`.
5. Consumer Pods scale up, process backlog, and acknowledge messages.
6. As lag drops below the threshold defined in the scaler definition and cooldown passes, KEDA scales consumer replicas back down.

## Diagram

![diagram](assets/nats-jetstream-keda-integration.drawio.svg)

## Deploy

1. Deploy the example resources:

   ```sh
   make deploy_app
   ```

2. Verify the setup job completed (creates stream and consumer):

   ```sh
   kubectl get jobs -n nats-jetstream-demo
   kubectl logs -n nats-jetstream-demo job/setup-jetstream
   ```

3. Verify workloads and scaler:

   ```sh
   kubectl get deploy,scaledobject,hpa -n nats-jetstream-demo
   ```

## Verify Scaling

1. Watch Pods and HPA:

   ```sh
   kubectl get pods -n nats-jetstream-demo -w
   kubectl get hpa -n nats-jetstream-demo -w
   ```

2. Check scaler and deployment state before reading logs:

   ```sh
   kubectl get scaledobject -n nats-jetstream-demo
   kubectl get deploy -n nats-jetstream-demo
   ```

   At idle, `jetstream-consumer` can be at 0 replicas. This is expected behavior.

3. Observe producer and consumer logs:

   ```console
   $ kubectl logs -n nats-jetstream-demo deploy/jetstream-producer
    [Producer] Publishing 100 messages every 90s
    12:01:40 Published 7 bytes to "orders.created"
    12:01:40 Published 7 bytes to "orders.created"
    12:01:41 Published 7 bytes to "orders.created"
    12:01:41 Published 7 bytes to "orders.created"
    12:01:41 Published 7 bytes to "orders.created"
    (...)
    12:01:44 Published 8 bytes to "orders.created"
    12:01:44 Published 8 bytes to "orders.created"
    12:01:44 Published 9 bytes to "orders.created"
    [Producer] Batch of 100 messages published
   ```

   ```console
   $ kubectl logs -n nats-jetstream-demo deploy/jetstream-consumer
    Found 3 pods, using pod/jetstream-consumer-54bb874699-kn2zg
    [Consumer] Processing 1 message every 1s
    [Consumer #1] Processed: order-307
    [Consumer #2] Processed: order-311
    [Consumer #3] Processed: order-313
    [Consumer #4] Processed: order-317
    [Consumer #5] Processed: order-322
    [Consumer #6] Processed: order-328
    (...)
   ```

4. Confirm scaling behavior:

- Producer sends 100 messages every 90 seconds.
- Consumer starts at 0 replicas.
- As lag grows, KEDA scales up the consumer deployment.
- After backlog is drained and cooldown passes, deployment scales back down.

Right after the demo app deployment, you should see a single producer and a successful initialization job:

```console
$ kubectl get pods -n nats-jetstream-demo
NAME                                  READY   STATUS      RESTARTS   AGE
jetstream-producer-86547dcdfd-mphdn   1/1     Running     0          28s
setup-jetstream-ptjfk                 0/1     Completed   0          28s
```

After some time you should see that a number of consumers increased:
```console
$ kubectl get pods -n nats-jetstream-demo
NAME                                  READY   STATUS      RESTARTS   AGE
jetstream-consumer-54bb874699-95gb2   1/1     Running     0          6s
jetstream-consumer-54bb874699-b6fm7   1/1     Running     0          37s
jetstream-consumer-54bb874699-jljwx   1/1     Running     0          6s
jetstream-consumer-54bb874699-psflk   1/1     Running     0          6s
jetstream-consumer-54bb874699-wjff6   1/1     Running     0          6s
jetstream-producer-86547dcdfd-mphdn   1/1     Running     0          68s
setup-jetstream-ptjfk                 0/1     Completed   0          68s

```

When the number of tasks falls below the defined threshold, the number of active consumers decreases.

## Cleanup

```sh
make undeploy_app
```