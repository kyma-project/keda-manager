#!/usr/bin/env bash

REF_NAME="${1:-"main"}"
RAW_EXPECTED_SHA=$(git log "${REF_NAME}" --max-count 1 --skip 1 --format=format:%H)
SHORT_EXPECTED_SHA=${RAW_EXPECTED_SHA:0:8}
DATE="v$(git log "${REF_NAME}" --max-count 1 --skip 1 --format=format:%ad --date=format:'%Y%m%d')"
EXPECTED_TAG="${DATE}-${SHORT_EXPECTED_SHA}"

IMAGE_TO_CHECK="${2:-europe-docker.pkg.dev/kyma-project/prod/keda-manager}"
BUMPED_IMAGE_TAG=$(cat sec-scanners-config.yaml | grep "${IMAGE_TO_CHECK}" | cut -d : -f 2)

if [[ "$BUMPED_IMAGE_TAG" != "$EXPECTED_TAG" ]]; then
  echo "Tags are not correct: wanted $EXPECTED_TAG but got $BUMPED_IMAGE_TAG"
  exit 1
fi
echo "Tags are correct"
exit 0
