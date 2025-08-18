#!/bin/sh

IMG_VERSION=${IMG_VERSION?"Define IMG_VERSION env"}

yq -i ".bdba[] |= sub(\"europe-docker.pkg.dev/kyma-project/prod/keda-manager:.*\", \"europe-docker.pkg.dev/kyma-project/prod/keda-manager:${IMG_VERSION}\")" sec-scanners-config.yaml
