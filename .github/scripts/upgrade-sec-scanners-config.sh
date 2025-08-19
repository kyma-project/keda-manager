#!/bin/sh

IMG_VERSION=${IMG_VERSION?"Define IMG_VERSION env"}

yq -i ".bdba[] |= select(contains(\"external\") | not) |= sub(\":.*\", \":${IMG_VERSION}\")" sec-scanners-config.yaml
