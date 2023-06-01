#!/usr/bin/env bash

# This script has the following arguments:
#     - link to a module image (required),
#     - credentials mode, allowed values (required):
#         dummy - dummy credentials passed
#         real - real credentials passed
#     - ci to indicate call from CI pipeline (optional)
# ./run_e2e_module_tests.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.0.0-PR-999 real ci
#
# The script requires the following environment variable set - these values are used to create unique SI and SB names:
#      GITHUB_RUN_ID - a unique number for each workflow run within a repository
#      GITHUB_JOB - the ID of the current job from the workflow
# The script requires the following environment variables if is called with "real" parameter - these should be real credentials base64 encoded:
#      SM_CLIENT_ID - client ID
#      SM_CLIENT_SECRET - client secret
#      SM_URL - service manager url
#      SM_TOKEN_URL - token url

#CI=${3-manual}  # if called from any workflow "ci" is expected here

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked


#[[ -z ${GITHUB_RUN_ID} ]] && echo "required variable GITHUB_RUN_ID not set" && exit 1

kubectl create namespace kyma-system
kubectl apply -f ../keda-manager.yaml

# check if deployment is available
while [[ $(kubectl get deployment/keda-manager -n kyma-system -o 'jsonpath={..status.conditions[?(@.type=="Available")].status}') != "True" ]];
do echo -e "\n---Waiting for deployment to be available"; sleep 5; done

echo -e "\n---Deployment available"

echo -e "\n---Installing Keda operator"
kubectl apply -f ../config/samples/operator_v1alpha1_keda.yaml

while [[ $(kubectl get keda/default -o 'jsonpath={..status.conditions[?(@.type=="Installed")].status}') != "True" ]];
do echo -e "\n---Waiting for Keda to be ready"; sleep 5; done

make -C ../hack/ci integration-test