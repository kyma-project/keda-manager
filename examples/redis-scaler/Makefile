deploy_app:
	kubectl apply -k k8s-resources

ensure_kyma_modules:
	kubectl create ns kyma-system
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-manager.yaml
	kubectl apply -f https://github.com/kyma-project/serverless-manager/releases/latest/download/serverless-operator.yaml 
	kubectl wait --for condition=established crd/kedas.operator.kyma-project.io
	kubectl apply -f https://github.com/kyma-project/keda-manager/releases/latest/download/keda-default-cr.yaml -n kyma-system
	kubectl wait --for condition=established crd/serverlesses.operator.kyma-project.io
	kubectl apply -f https://github.com/kyma-project/serverless-manager/releases/latest/download/default-serverless-cr.yaml -n kyma-system