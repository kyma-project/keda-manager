#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
PULL_BASE_REF=${PULL_BASE_REF?"Define PULL_BASE_REF env"} # name of the tag
GITHUB_TOKEN=${GITHUB_TOKEN?"Define GITHUB_TOKEN env"} # github token used to upload the template yaml
RELEASE_ID=${RELEASE_ID?"Define RELEASE_ID env"} # github token used to upload the template yaml

uploadFile() {
  filePath=${1}
  ghAsset=${2}

  echo "Uploading ${filePath} as ${ghAsset}"
  response=$(curl -s -o output.txt -w "%{http_code}" \
                  --request POST --data-binary @"$filePath" \
                  -H "Authorization: token $GITHUB_TOKEN" \
                  -H "Content-Type: text/yaml" \
                   $ghAsset)
  if [[ "$response" != "201" ]]; then
    echo "Unable to upload the asset ($filePath): "
    echo "HTTP Status: $response"
    cat output.txt
    exit 1
  else
    echo "$filePath uploaded"
  fi
}

echo "PULL_BASE_REF ${PULL_BASE_REF}"

MODULE_VERSION=${PULL_BASE_REF} make render-manifest

echo "Generated keda-manager.yaml:"
cat keda-manager.yaml

echo "Updating github release with assets"
UPLOAD_URL="https://uploads.github.com/repos/kyma-project/keda-manager/releases/${RELEASE_ID}/assets"

uploadFile "keda-manager.yaml" "${UPLOAD_URL}?name=keda-manager.yaml"
uploadFile "config/samples/keda-default-cr.yaml" "${UPLOAD_URL}?name=keda-default-cr.yaml"
