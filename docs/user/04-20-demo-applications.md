## Keda CPU Scaler Example

## Context
This demo application shows how to scale the Kubernetes workloads using KEDA API based on a simple CPU consumption case.

See the KEDA documentation:
 - [API of Scaled Object Custom Resource](https://keda.sh/docs/latest/concepts/scaling-deployments/#scaledobject-spec)
 - [Available scalers](https://keda.sh/docs/latest/scalers/)

The KEDA demo application consists of:
 - order-service deployment (serving as a scale target)
 - busybox deployment (generating trafic)
 - scaled object using a simple CPU-based trigger

## Procedure

1. Deploy the demo application:

```bash
kubectl apply -f examples/keda-cpu-scaler-demo.yml
```

2. Verify the successful deployment of the demo application:

You should see that scaled object is created and has a status READY:

```bash
kunectl get scaledobjects.keda.sh -n keda-demo
NAME                        SCALETARGETKIND      SCALETARGETNAME   MIN   MAX   TRIGGERS   AUTHENTICATION   READY   ACTIVE   FALLBACK   AGE
orders-service-cpu-scaler   apps/v1.Deployment   orders-service    1     10    cpu                         True    True     Unknown    8m3s
```

You should also see that after a while, KEDA has engaged the Kubernetes HorizontalPodAutoscaler, which controls the number of replicas of the target deployment.

```bash
kubectl get hpa -n keda-demo
NAMESPACE   NAME                                 REFERENCE                   TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
keda-demo   keda-hpa-orders-service-cpu-scaler   Deployment/orders-service   80%/30%   1         10        4          31s
```

## Keda Prometheus Scaler Example

Follow [this example](https://github.com/kyma-project/examples/tree/main/scale-to-zero-with-keda) to experience how Kyma's Keda module can complement other Kyma components.

It demonstrates an event-driven approach that allows you to decouple functional parts of an application and apply consumption-based scaling.

It uses: 
 - Functions to deploy workloads directly from a Git repository ([Kyma Serverless](https://kyma-project.io/docs/kyma/latest/01-overview/serverless/)),
 - In-cluster Eventing to enable event-driven communication ([Kyma Eventing](https://kyma-project.io/docs/kyma/latest/01-overview/eventing/)), 
 - Prometheus and Istio to deliver metrics essential for scaling decisions,
 - Keda to drive the scaling.

![scenario](../assets/scaling-scenario.png "Scenario")