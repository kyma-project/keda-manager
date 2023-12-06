#!/usr/bin/env bash

DESIRED_TAG=$1

IMAGE_TO_CHECK="${2:-europe-docker.pkg.dev/kyma-project/prod/keda-manager}"
BUMPED_IMAGE_TAG=$(cat sec-scanners-config.yaml | grep "${IMAGE_TO_CHECK}" | cut -d : -f 2)

if [[ "$BUMPED_IMAGE_TAG" != "$DESIRED_TAG" ]]; then
  echo "Tags are not correct: wanted $DESIRED_TAG but got $BUMPED_IMAGE_TAG"
  exit 1
fi
echo "Tags are correct"
exit 0
