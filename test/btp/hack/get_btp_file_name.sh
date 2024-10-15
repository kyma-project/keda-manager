#!/bin/bash

function get_btp_file_name () {

  local _OS_TYPE=$1
  local _OS_ARCH=$2

  [ "$_OS_TYPE" == "Linux"   ] && [ "$_OS_ARCH" == "x86_64" ] && echo "btp-cli-linux-amd64-latest.tar.gz"   ||
  [ "$_OS_TYPE" == "Linux"   ] && [ "$_OS_ARCH" == "arm64"  ] && echo "btp-cli-linux-arm64-latest.tar.gz"   ||
  [ "$_OS_TYPE" == "Darwin"  ] && [ "$_OS_ARCH" == "x86_64" ] && echo "btp-cli-darwin-amd64-latest.tar.gz"  ||
  [ "$_OS_TYPE" == "Darwin"  ] && [ "$_OS_ARCH" == "arm64"  ] && echo "btp-cli-darwin-arm64-latest.tar.gz"
}

get_btp_file_name "$@"
