#!/usr/bin/env bash

REPOSITORY=kyma-project/keda-manager
GITHUB_URL=https://api.github.com/repos/${REPOSITORY}/commits/main

COMMIT_HASH=$(curl -sS "$GITHUB_URL" | grep sha | head -n 1 | tr -d "\ " | cut -d : -f 2 | tr -d "," | tr -d '"')
echo $COMMIT_HASH

RAW_HASH=$(git log -n 1 | grep commit | tr -d "commit")
COMMIT_HASH2=${RAW_HASH:0:9}
echo $COMMIT_HASH2

SECURITY_IMAGE=$(cat sec-scanners-config.yaml | grep europe-docker.pkg.dev/kyma-project/prod/keda-manager | cut -d : -f 2 | cut -d - -f 2)
echo $SECURITY_IMAGE

if [[ "$SECURITY_IMAGE" != "$COMMIT_HASH2" ]]; then
  echo "Tags are not correct: wanted $DESIRED_TAG but got $MODULE_VERSION"
  exit 1
fi

exit 0

#git rev-parse --sh