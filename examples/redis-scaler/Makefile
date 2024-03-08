deploy_app:
	(cd src/emitter-fn && kyma apply function --dry-run -oyaml > ../../k8s-resources/emitter-fn.yaml)
	kubectl apply -k k8s-resources