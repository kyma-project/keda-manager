#!/bin/sh
set -e

USERNAME=$(kubectl get secrets -n kyma-system dockerregistry-config-external -o jsonpath={.data.username} --kubeconfig ${KUBECONFIG} | base64 -d)
PASSWORD=$(kubectl get secrets -n kyma-system dockerregistry-config-external -o jsonpath={.data.password} --kubeconfig ${KUBECONFIG} | base64 -d)
REGISTRY_URL=$(kubectl get dockerregistries.operator.kyma-project.io -n kyma-system default -ojsonpath={.status.externalAccess.pushAddress} --kubeconfig ${KUBECONFIG})

echo $USERNAME
echo $PASSWORD

echo Testing Docker Registry availibility at: $REGISTRY_URL

COUNTER=0
RESPONSE_CODE=$(curl  -o /dev/null -u $USERNAME:$PASSWORD -L -w ''%{http_code}'' --connect-timeout 5 \
    --max-time 10 \
    --retry 5 \
    --retry-delay 0 \
    --retry-max-time 40 $REGISTRY_URL )
echo Response from registry: $RESPONSE_CODE
if [ $RESPONSE_CODE == '200' ]; then
    exit 0
fi
until [[ $COUNTER -gt 10  ||  $RESPONSE_CODE == "200" ]] ;
do
    sleep 5
    let COUNTER=COUNTER+1 
    RESPONSE_CODE=$(curl -s -o /dev/null -u $USERNAME:$PASSWORD -L -w ''%{http_code}'' --connect-timeout 5 \
    --max-time 10 \
    --retry 5 \
    --retry-delay 0 \
    --retry-max-time 40 $REGISTRY_URL )
    echo Response from registry: $RESPONSE_CODE
    if [ $RESPONSE_CODE == '200' ]; then
        exit 0
    fi
done

echo "ERROR"
exit 1
