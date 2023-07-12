#!/usr/bin/env bash

RAW_EXPECTED_HASH=$(git log main --author "Kyma Bot" --max-count 1 --skip 1 --format=format:%H)
SHORT_EXPECTED_HASH=${RAW_EXPECTED_HASH:0:8}
DATE="v$(git log main --author "Kyma Bot" --max-count 1 --skip 1 --format=format:%ad --date=format:'%Y%m%d')"
EXPECTED_HASH="${DATE}-${SHORT_EXPECTED_HASH}"

IMAGE_TO_CHECK="${1:-europe-docker.pkg.dev/kyma-project/prod/keda-manager}"
BUMPED_IMAGE_HASH=$(cat sec-scanners-config.yaml | grep "${IMAGE_TO_CHECK}" | cut -d : -f 2)

if [[ "$BUMPED_IMAGE_HASH" != "$EXPECTED_HASH" ]]; then
  echo "Tags are not correct: wanted $EXPECTED_HASH but got $BUMPED_IMAGE_HASH"
  exit 1
fi
echo "Tags are correct"
exit 0
