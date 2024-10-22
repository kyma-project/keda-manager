#!/bin/bash

get_all_and_fail() {
	kubectl get all --all-namespaces
	exit 1
}

echo "waiting for deployment"
kubectl wait -n kyma-system --for=condition=Available --timeout=2m deployment keda-manager || get_all_and_fail

echo "waiting for pod"
kubectl wait -n kyma-system --for=condition=Ready --timeout=2m pod --selector "app.kubernetes.io/component"="keda-manager.kyma-project.io" || get_all_and_fail

echo "waiting for keda"
kubectl wait -n kyma-system --for=condition=Installed --timeout=2m keda default || get_all_and_fail
