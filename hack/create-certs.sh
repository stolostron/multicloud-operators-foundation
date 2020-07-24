#!/bin/bash

set -o errexit

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
SED="sed"
if [ "${OS}" == "darwin" ]; then
    SED="gsed"
    if [ ! -x "$(command -v ${SED})"  ]; then
       echo "ERROR: $SED required, but not found."
       echo "Perform \"brew install gnu-sed\" and try again."
       exit 1
    fi
fi

DIRPATH="$(cd "$(dirname "$0")"; pwd)"
DEPLOYPAHT="${DIRPATH}/../deploy/dev/hub/resources"

# create certs
go mod tidy
go run hack/certs/cert.go --path="${DIRPATH}"

# overrides the manifests
acm_agent_ca=$(cat "${DIRPATH}"/acm-agent-ca.pem)
acm_agent_client=$(cat "${DIRPATH}"/acm-agent-client.pem)
acm_agent_key=$(cat "${DIRPATH}"/acm-agent-key.pem)
acm_apiserver_ca=$(cat "${DIRPATH}"/acm-apiserver-ca.pem)
acm_apiserver_client=$(cat "${DIRPATH}"/acm-apiserver-client.pem)
acm_apiserver_key=$(cat "${DIRPATH}"/acm-apiserver-key.pem)

${SED} -i "s/\$ACM_AGENT_CA/$acm_agent_ca/g" "${DEPLOYPAHT}"/100-agent-cert.yaml
${SED} -i "s/\$ACM_AGENT_CLIENT/$acm_agent_client/g" "${DEPLOYPAHT}"/100-agent-cert.yaml
${SED} -i "s/\$ACM_AGENT_KEY/$acm_agent_key/g" "${DEPLOYPAHT}"/100-agent-cert.yaml
${SED} -i "s/\$ACM_APISERVER_CA/$acm_apiserver_ca/g" "${DEPLOYPAHT}"/100-apiserver-cert.yaml
${SED} -i "s/\$ACM_APISERVER_CLIENT/$acm_apiserver_client/g" "${DEPLOYPAHT}"/100-apiserver-cert.yaml
${SED} -i "s/\$ACM_APISERVER_KEY/$acm_apiserver_key/g" "${DEPLOYPAHT}"/100-apiserver-cert.yaml
${SED} -i "s/\$ACM_APISERVER_CA/$acm_apiserver_ca/g" "${DEPLOYPAHT}"/100-proxyserver-apiservice.yaml

# clean cert files
rm "${DIRPATH}"/*.pem
