#!/usr/bin/env bash

set -e

DESIRED_TAG=$1

RAW_MODULE_VERSION=$(cat .env | grep MODULE_VERSION | grep -v ifndef)
MODULE_VERSION=$(echo $RAW_MODULE_VERSION |  tr -d '\ ' | cut -d = -f 2)

if [[ "$DESIRED_TAG" != "$MODULE_VERSION" ]]; then
  echo "Tags are not correct: wanted $DESIRED_TAG but got $MODULE_VERSION"
  exit 1
fi
echo "Tags are correct"
exit 0
