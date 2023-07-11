#!/usr/bin/env bash

set -e

DESIRED_TAG=$1

source .version

if [[ "$DESIRED_TAG" != "$MODULE_VERSION" ]]; then
  echo "Tags are not correct: wanted $DESIRED_TAG but got $MODULE_VERSION"
  exit 1
fi
echo "Tags are correct"
exit 0
