deploy_app:
	kubectl apply -k k8s-resources

ensure_kyma_modules:
	kubectl get ns kyma-system >/dev/null 2>&1 || kubectl create ns kyma-system
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
	kubectl wait --for condition=established --timeout=180s crd/kedas.operator.kyma-project.io
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-default-cr.yaml -n kyma-system
	kubectl apply -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-manager.yaml
	kubectl wait --for condition=established --timeout=180s crd/nats.operator.kyma-project.io
	kubectl apply -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-default-cr.yaml -n kyma-system

undeploy_app:
	kubectl delete -k k8s-resources