## Deploy 

### Prerequisites

 - `KUBECONFIG` env variable pointing to a kubernetes instance
 - helm client installed locally in your machine
 - keda module https://github.com/kyma-project/keda-manager?tab=readme-ov-file#install-keda-manager-and-keda-from-the-latest-release in your kyma cluster 
 - serverless module https://github.com/kyma-project/serverless?tab=readme-ov-file#install in your kyma cluster 


### Deploy redis

Install redis into `redis` namespace.

```sh
kubectl create ns redis
```

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
kubectl create namespace redis
helm install my-release bitnami/redis --namespace redis
```

### Deploy example

```sh
make deploy_app
```

```sh
scaledjob.keda.sh/processor-scaled-job created
function.serverless.kyma-project.io/emitter-fn created
```

### Verify

Observe pods in redis namespace. There is one function pod and redis itself.
```sh
kubectl get pods -n redis -w
NAME                                READY   STATUS        RESTARTS        AGE
my-release-redis-master-0           1/1     Running       0               9h
my-release-redis-replicas-1         1/1     Running       0               9h
my-release-redis-replicas-2         1/1     Running       5 (6h50m ago)   21h
my-release-redis-replicas-0         1/1     Running       5 (6h50m ago)   21h
emitter-fn-build-j46wc-6sx6v        0/1     Completed     0               14m
emitter-fn-nkzkg-78cd47cd78-lxbms   1/1     Running       1 (13m ago)     14m
```

Expose emitter Function via port-forward and send a few messages to populate redis list.

```sh
kubectl port-forward -n redis svc/emitter-fn  8080:80
```

Send a few messages via POST request towards emitter-fn

```sh
curl -H "Content-Type: application/json" -X POST -d '{"msg":"hello1"}' localhost:8080
curl -H "Content-Type: application/json" -X POST -d '{"msg":"hello2"}' localhost:8080
curl -H "Content-Type: application/json" -X POST -d '{"msg":"hello3"}' localhost:8080
```

Keda should spin scaled jobs as a result to new messages in the redis list. They will do their job and enter `Completed` state.

```sh
kubecttl get pods -n redis -w 
NAME                                READY   STATUS        RESTARTS      AGE
emitter-fn-g9g8j-7f4f74854f-mq2nq   0/1     Terminating   0             21h
my-release-redis-master-0           1/1     Running       0             10h
my-release-redis-replicas-1         1/1     Running       0             10h
my-release-redis-replicas-2         1/1     Running       5 (7h ago)    21h
my-release-redis-replicas-0         1/1     Running       5 (7h ago)    21h
emitter-fn-nkzkg-78cd47cd78-lxbms   1/1     Running       1 (23m ago)   24m
processor-scaled-job-66jjn-wjgjr    0/1     Completed     0             12s
processor-scaled-job-n68tp-x2p64    0/1     Pending       0             0s
processor-scaled-job-n68tp-x2p64    0/1     Pending       0             0s
processor-scaled-job-vn4lg-g694x    0/1     Pending       0             0s
processor-scaled-job-vn4lg-g694x    0/1     Pending       0             0s
processor-scaled-job-n68tp-x2p64    0/1     ContainerCreating   0             0s
processor-scaled-job-vn4lg-g694x    0/1     ContainerCreating   0             0s
processor-scaled-job-vn4lg-g694x    1/1     Running             0             2s
processor-scaled-job-n68tp-x2p64    1/1     Running             0             2s
processor-scaled-job-n68tp-x2p64    0/1     Completed           0             3s
processor-scaled-job-vn4lg-g694x    0/1     Completed           0             3s
processor-scaled-job-vn4lg-g694x    0/1     Completed           0             5s
processor-scaled-job-n68tp-x2p64    0/1     Completed           0             5s
processor-scaled-job-vn4lg-g694x    0/1     Completed           0             6s
processor-scaled-job-n68tp-x2p64    0/1     Completed           0             6s
```

By inspecting logs of the jobs you should see all messages were processed
```sh
kubectl logs -n redis -l scaledjob.keda.sh/name=processor-scaled-job -f
Processing started for hello2.. will finish in 8283ms
Processing started for hello3.. will finish in 1523ms
Processing started for hello1.. will finish in 8969ms
```
