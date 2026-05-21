#!/bin/sh

IMG_VERSION=${IMG_VERSION?"Define IMG_VERSION env"}

yq -i ".images[] |= select((contains(\"external\") or contains(\"restricted-prod\")) | not) |= sub(\":.*\", \":${IMG_VERSION}\")" component-config.yaml
