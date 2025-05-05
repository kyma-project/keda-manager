#!/bin/sh

IMG_VERSION=${IMG_VERSION?"Define IMG_VERSION env"}

yq -i ".bdba[] |= sub(\":main\", \":${IMG_VERSION}\")" sec-scanners-config.yaml
