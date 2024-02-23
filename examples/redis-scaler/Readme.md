## Deploy 

### Prerequisites

 - `KUBECONFIG` env variable pointing to a kubernetes instance
 - helm installed
 - keda module installed
 - serverless module installed


### Deploy redis

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
kubectl create namespace redis
helm install my-release bitnami/redis --namespace redis
```

### Configure redis

Verify `k8s-resources/redis.env`. Make sure `REDIS_HOST` env points to the kubernetes local service url of redis.

### Deploy example

```sh
kubectl apply -k k8s-resources
```

### Verify

You should see that scaled object is active
```sh
kubectl get scaledobjects.keda.sh -n redis
NAME                         SCALETARGETKIND      SCALETARGETNAME   MIN   MAX   TRIGGERS   AUTHENTICATION   READY   ACTIVE   FALLBACK   PAUSED    AGE
processor-fn-scaled-object   apps/v1.Deployment   processor         0     5     redis                       True    False    Unknown    Unknown   6m27s
```

For empty redis list you should see that keda scaled down `processor` deployment to zero.

```sh
kubectl get hpa -n redis
NAME                                  REFERENCE              TARGETS             MINPODS   MAXPODS   REPLICAS   AGE
keda-hpa-processor-fn-scaled-object   Deployment/processor   <unknown>/5 (avg)   1         5         0          4m49s
```