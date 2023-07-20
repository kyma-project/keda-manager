#!/usr/bin/env bash

RAW_EXPECTED_TAG=$(git log main --max-count 1 --skip 1 --format=format:%H)
SHORT_EXPECTED_TAG=${RAW_EXPECTED_TAG:0:8}
DATE="v$(git log main --max-count 1 --skip 1 --format=format:%ad --date=format:'%Y%m%d')"
EXPECTED_TAG="${DATE}-${SHORT_EXPECTED_TAG}"

IMAGE_TO_CHECK="${1:-europe-docker.pkg.dev/kyma-project/prod/keda-manager}"
BUMPED_IMAGE_TAG=$(cat sec-scanners-config.yaml | grep "${IMAGE_TO_CHECK}" | cut -d : -f 2)

if [[ "$BUMPED_IMAGE_TAG" != "$EXPECTED_TAG" ]]; then
  echo "Tags are not correct: wanted $EXPECTED_TAG but got $BUMPED_IMAGE_TAG"
  exit 1
fi
echo "Tags are correct"
exit 0
