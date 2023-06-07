#!/usr/bin/env bash

export ACCESS_TOKEN=$1

echo "Checking status of POST Jobs for Keda-Manager"

if curl -L -H "Accept: application/vnd.github+json" -H "Authorization: Bearer ${GITHUB_TOKEN}" -H "X-GitHub-Api-Version: 2022-11-28" https://api.github.com/repos/kyma-project/keda-manager/commits/main/status | grep -q 'failure\|pending'
then
  echo "Jobs failed and/or in progress - check post jobs status"
  exit 1
else
  echo "All jobs succeeded"
fi