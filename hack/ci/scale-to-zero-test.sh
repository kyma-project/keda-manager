#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="${1:?PROJECT_ROOT is required}"
TEST_DOMAIN="${2:?TEST_DOMAIN is required}"
TIMEOUT=120
SLEEP_INTERVAL=5

kubectl create ns demo-app || true
kubectl label namespace demo-app istio-injection=enabled --overwrite

sed "s/\${TEST_DOMAIN}/${TEST_DOMAIN}/g" "${PROJECT_ROOT}/examples/scale-to-zero-with-keda/k8s-resources/apirule.yaml" | kubectl apply -f -
sed "s/\${TEST_DOMAIN}/${TEST_DOMAIN}/g" "${PROJECT_ROOT}/examples/scale-to-zero-with-keda/k8s-resources/envoyfilter.yaml" | kubectl apply -f -
sed "s/\${TEST_DOMAIN}/${TEST_DOMAIN}/g" "${PROJECT_ROOT}/examples/scale-to-zero-with-keda/k8s-resources/httpscaledobject.yaml" | kubectl apply -f -
kubectl apply -f "${PROJECT_ROOT}/examples/scale-to-zero-with-keda/k8s-resources/demo-app.yaml"

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

# Send request to trigger scale-up
echo "Sending request to trigger scale-up..."
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "Content-Type: application/json" -X GET -d '{"foo":"bar"}' "https://http-echo-keda.${TEST_DOMAIN}")
echo "HTTP status: ${HTTP_STATUS}"
if [ "${HTTP_STATUS}" != "200" ]; then
	echo "Expected HTTP 200, got ${HTTP_STATUS}"
	exit 1
fi

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

# Wait for http-echo to scale back to zero
echo "Waiting for http-echo to scale back to 0..."
for i in $(seq 1 ${TIMEOUT}); do
	REPLICAS=$(kubectl get deployment http-echo -n demo-app -o jsonpath='{.spec.replicas}' 2>/dev/null)
	if [ "$REPLICAS" = "0" ]; then
		echo "http-echo scaled back to 0"
		break
	fi
	if [ "$i" -eq "${TIMEOUT}" ]; then
		echo "Timed out waiting for scale back to zero"
		exit 1
	fi
	sleep ${SLEEP_INTERVAL}
done

echo "Scale-to-zero test passed"

