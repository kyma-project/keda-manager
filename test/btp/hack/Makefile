.PHONY: service-account-kubeconfig
service-account-kubeconfig:
	@echo "Generating expiring service-account based kubeconfig..."
	../bin/kyma alpha access --clusterrole ${CLUSTERROLE} --kubeconfig ../tf/kubeconfig.yaml --name ci-workflow-sa --time 1h --output tmp/sa-kubeconfig.yaml
	@echo "Service-account based kubeconfig generated"

.PHONY cleanup-tf-force:
cleanup-tf-force:
	rm -rf ../tf/*.txt
	rm -rf ../tf/*.json
	rm -rf ../tf/*.env
	rm -rf ../tf/*.txt
	rm -rf ../tf/terraform.state
	rm -rf ../tf/terraform.state
	rm -rf ../tf/terraform.tfstate
	rm -rf ../tf/terraform.tfstate*
	rm -rf ../tf/.terraform.locl.hcl
