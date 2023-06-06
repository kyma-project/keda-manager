##!/usr/bin/env bash
#
## This script has the following arguments:
##     - link to a module image (required),
##     - ci to indicate call from CI pipeline (optional)
##
## The script requires the following environment variable set - these values are used to create unique SI and SB names:
##      GITHUB_RUN_ID - a unique number for each workflow run within a repository
##      GITHUB_JOB - the ID of the current job from the workflow
#
#CI=${3-manual}  # if called from any workflow "ci" is expected here
#
## standard bash error handling
#set -o nounset  # treat unset variables as an error and exit immediately.
#set -o errexit  # exit immediately when a command fails.
#set -E          # needs to be set if we want the ERR trap
#set -o pipefail # prevents errors in a pipeline from being masked
#
#MODULE_IMAGE_NAME=$1
#YAML_DIR="scripts/testing/yaml"
#
#
#curl -LO https://github.com/kyma-project/keda-manager/releases/download/${MODULE_IMAGE_NAME}/keda-manager.yaml
#
#[[ -z ${GITHUB_RUN_ID} ]] && echo "required variable GITHUB_RUN_ID not set" && exit 1
#
#kubectl create namespace kyma-system
kubectl apply -f keda-manager.yaml

# check if deployment is available
while [[ $(kubectl get deployment/keda-manager -n kyma-system -o 'jsonpath={..status.conditions[?(@.type=="Available")].status}') != "True" ]];
do echo -e "\n---Waiting for deployment to be available"; sleep 5; done

echo -e "\n---Deployment available"

echo -e "\n---Installing Keda operator"
kubectl apply -f config/samples/operator_v1alpha1_keda.yaml

while [[ $(kubectl get keda/default -o 'jsonpath={..status.conditions[?(@.type=="Installed")].status}') != "True" ]];
do echo -e "\n---Waiting for Keda to be ready"; sleep 5; done

make -C hack/ci integration-test

kubectl delete kedas.operator.kyma-project.io default
while [[ $(kubectl get keda/default -o 'jsonpath={..status.conditions[?(@.type=="Installed")].status}') = "True" ]];
do echo -e "\n---Waiting for Keda to be deleted"; sleep 5; done
