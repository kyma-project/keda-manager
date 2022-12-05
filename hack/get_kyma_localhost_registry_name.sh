#!/bin/bash
set -e

function get_kyma_localhost_registry_name () {
	local _REGISTRY_NAME=$1
	local _TMPFILE=/tmp/coredns.patch.yaml
	local _COREDNS_RDY=0
	local _NUMBER=1

	while [[ $_NUMBER -le 12*1 ]] ; do
		echo ">--> waiting to patch coredns #$_NUMBER"
		_COREDNS_RDY=$(kubectl get cm \
			-n kube-system coredns \
			-o jsonpath='{.data.NodeHosts}'\
	        | grep -e ' k3d-kyma-registry$' \
		| wc -l \
		| xargs) \
	       || 0
		[[ "$_COREDNS_RDY" == 1 ]] && echo 'coredns rdy to patch' && break
		sleep 5
		((_NUMBER = _NUMBER + 1))
	done

	[[ "$_COREDNS_RDY" == 0 ]] && echo '### timeout reached - unable to patch coredns' && return 1

	local _LOCAL_REGISTRY_NAME=$(kubectl get cm \
		-n kube-system \
		coredns \
		-o yaml \
	      | tee ${_TMPFILE} \
	      | yq '.data.NodeHosts' \
	      | grep -e " ${_REGISTRY_NAME}$" \
	      | sed "s/${_REGISTRY_NAME}/${_REGISTRY_NAME}.localhost/g")
	
	yq -i ".data.NodeHosts += \"$_LOCAL_REGISTRY_NAME\"" ${_TMPFILE}
	kubectl patch -n kube-system cm coredns --patch-file ${_TMPFILE}
}

get_kyma_localhost_registry_name "$@"
