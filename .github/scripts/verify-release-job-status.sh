#!/usr/bin/env bash



echo "Checking status of POST Jobs for Keda-Manager"

REF_NAME="${1:-"main"}"
STATUS_URL="https://api.github.com/repos/kyma-project/keda-manager/commits/${REF_NAME}/status"

function get_keda_status () {
	local number=1
	while [[ $number -le 100 ]] ; do
		echo ">--> checking keda release job status #$number"
		local STATUS=`curl -L -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" ${STATUS_URL} | head -n 2 `
		echo "jobs status: ${STATUS:='UNKNOWN'}"
		[[ "$STATUS" == *"Success"* ]] && return 0
		sleep 5
        	((number = number + 1))
	done

	exit 1
}

get_kyma_status
