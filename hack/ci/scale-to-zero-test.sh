#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="${1:?PROJECT_ROOT is required}"
MODE="${2:-local}"
TIMEOUT=420
SLEEP_INTERVAL=5

setup_local() {
    HOST="http-echo.demo-app.svc.cluster.local"

    kubectl create ns demo-app || true
    kubectl apply -f "${PROJECT_ROOT}/examples/scale-to-zero-with-keda/k8s-resources/demo-app.yaml"

    cat <<EOF | kubectl apply -f -
apiVersion: http.keda.sh/v1alpha1
kind: HTTPScaledObject
metadata:
  name: http-echo
  namespace: demo-app
spec:
  hosts:
  - "${HOST}"
  pathPrefixes:
  - /
  scaleTargetRef:
    name: http-echo
    kind: Deployment
    apiVersion: apps/v1
    service: http-echo
    port: 8080
  replicas:
    min: 0
    max: 10
  scalingMetric:
    requestRate:
      targetValue: 10
      granularity: "1s"
      window: "1m"
EOF
}

setup_btp() {
    TEST_DOMAIN="${3:?TEST_DOMAIN is required for btp mode}"

    kubectl create ns demo-app || true
    kubectl label namespace demo-app istio-injection=enabled --overwrite

    sed "s/\${TEST_DOMAIN}/${TEST_DOMAIN}/g" "${PROJECT_ROOT}/examples/scale-to-zero-with-keda/k8s-resources/apirule.yaml" | kubectl apply -f -
    sed "s/\${TEST_DOMAIN}/${TEST_DOMAIN}/g" "${PROJECT_ROOT}/examples/scale-to-zero-with-keda/k8s-resources/envoyfilter.yaml" | kubectl apply -f -
    sed "s/\${TEST_DOMAIN}/${TEST_DOMAIN}/g" "${PROJECT_ROOT}/examples/scale-to-zero-with-keda/k8s-resources/httpscaledobject.yaml" | kubectl apply -f -
    kubectl apply -f "${PROJECT_ROOT}/examples/scale-to-zero-with-keda/k8s-resources/demo-app.yaml"
}

send_request_local() {
    echo "Sending request via interceptor-proxy to trigger scale-up..."
    kubectl port-forward -n kyma-system svc/keda-add-ons-http-interceptor-proxy 8888:8080 &
    PF_PID=$!
    sleep 2

    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "Host: ${HOST}" -H "Content-Type: application/json" -X GET -d '{"foo":"bar"}' "http://localhost:8888" --max-time 30 || true)
    echo "HTTP status: ${HTTP_STATUS}"

    kill $PF_PID 2>/dev/null || true

    if [ "${HTTP_STATUS}" != "200" ]; then
        echo "Expected HTTP 200, got ${HTTP_STATUS}"
        echo "Checking interceptor logs..."
        kubectl logs -n kyma-system -l app.kubernetes.io/instance=interceptor --tail=20
        exit 1
    fi
}

send_request_btp() {
    TEST_DOMAIN="${3}"
    echo "Sending request to trigger scale-up..."
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "Content-Type: application/json" -X GET -d '{"foo":"bar"}' "https://http-echo-keda.${TEST_DOMAIN}")
    echo "HTTP status: ${HTTP_STATUS}"
    if [ "${HTTP_STATUS}" != "200" ]; then
        echo "Expected HTTP 200, got ${HTTP_STATUS}"
        exit 1
    fi
}

# Setup
case "${MODE}" in
    local) setup_local ;;
    btp)   setup_btp "$@" ;;
    *)     echo "Unknown mode: ${MODE}. Use 'local' or 'btp'"; exit 1 ;;
esac

# Wait for http-echo to scale to zero
echo "Waiting for http-echo to scale to 0 replicas..."
for i in $(seq 1 ${TIMEOUT}); do
    REPLICAS=$(kubectl get deployment http-echo -n demo-app -o jsonpath='{.spec.replicas}' 2>/dev/null)
    if [ "$REPLICAS" = "0" ]; then
        echo "http-echo scaled to 0"
        break
    fi
    if [ "$i" -eq "${TIMEOUT}" ]; then
        echo "Timed out waiting for scale to zero"
        kubectl get httpscaledobjects -n demo-app -o yaml 2>/dev/null || true
        kubectl get deployment http-echo -n demo-app -o yaml 2>/dev/null || true
        exit 1
    fi
    sleep ${SLEEP_INTERVAL}
done

# Wait for http-echo pods to be fully terminated
echo "Waiting for http-echo pods to terminate..."
for i in $(seq 1 ${TIMEOUT}); do
    POD_COUNT=$(kubectl get pods -n demo-app -l app=http-echo --no-headers 2>/dev/null | wc -l | tr -d ' ')
    if [ "$POD_COUNT" = "0" ]; then
        echo "All http-echo pods terminated"
        break
    fi
    if [ "$i" -eq "${TIMEOUT}" ]; then
        echo "Timed out waiting for pods to terminate"
        exit 1
    fi
    sleep ${SLEEP_INTERVAL}
done

# Send request
case "${MODE}" in
    local) send_request_local ;;
    btp)   send_request_btp "$@" ;;
esac

# Wait for http-echo to scale up
echo "Waiting for http-echo to scale up..."
for i in $(seq 1 ${TIMEOUT}); do
    REPLICAS=$(kubectl get deployment http-echo -n demo-app -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
    if [ -n "$REPLICAS" ] && [ "$REPLICAS" -ge 1 ]; then
        echo "http-echo scaled up to ${REPLICAS} replicas"
        break
    fi
    if [ "$i" -eq "${TIMEOUT}" ]; then
        echo "Timed out waiting for scale up"
        exit 1
    fi
    sleep ${SLEEP_INTERVAL}
done

# Schedule cleanup in the background and pass immediately
echo "Scheduling background cleanup of demo-app namespace..."
kubectl delete ns demo-app --wait=false &

echo "Scale-to-zero test passed (mode: ${MODE})"
