#!/bin/bash

set -xv
set -o nounset
set -o pipefail

pwd

HELM=${HELM:-_output/tools/bin/helm}

KUBECTL=${KUBECTL:-kubectl}
OCM_BRANCH=${OCM_BRANCH:-main}

IMAGE_CLUSTER_PROXY=${IMAGE_CLUSTER_PROXY:-quay.io/stolostron/cluster-proxy}
IMAGE_MANAGED_SERVICEACCOUNT=${IMAGE_MANAGED_SERVICEACCOUNT:-quay.io/stolostron/managed-serviceaccount}


function waitForAddon() {
  FOUND=1
  MINUTE=0
  addonName=$1
  clusterName=$2

  echo "\n#####\nWait for ${addonName} to reach available state (2min).\n"
  while [ ${FOUND} -eq 1 ]; do
          # Wait up to 4min, should only take about 20-30s
          if [ $MINUTE -gt 120 ]; then
              echo "Timeout waiting for the ${addonName}. "
              echo "List of current addon controller pods:"
              oc -n open-cluster-management get pods
              echo
              exit 1
          fi

          operatorAddon=`oc -n ${clusterName} get mca | grep ${addonName}`

          if [[ $(echo $operatorAddon | grep "${addonName}") ]]; then
              echo "* ${addonName} is created"
              break
          fi

          sleep 3
          (( MINUTE = MINUTE + 3 ))
      done
}


# Use the same chart that installer used in the backplane-operator repo
rm -rf backplane-operator
echo "############  Cloning backplane-operator"
git clone --depth 1 --branch "$OCM_BRANCH" https://github.com/stolostron/backplane-operator.git

cd backplane-operator || {
  printf "cd failed, backplane-operator does not exist"
  exit 1
}

BASEDDOMAIN=$($KUBECTL get ingress.config.openshift.io cluster -o=jsonpath='{.spec.domain}')

# Install cluster-proxy CRDs first
oc apply -f https://raw.githubusercontent.com/stolostron/cluster-proxy/main/charts/cluster-proxy/crds/managedproxyconfigurations.yaml
oc apply -f https://raw.githubusercontent.com/stolostron/cluster-proxy/main/charts/cluster-proxy/crds/managedproxyserviceresolvers.yaml

../$HELM install \
	-n open-cluster-management --create-namespace \
	cluster-proxy-addon pkg/templates/charts/toggle/cluster-proxy-addon \
  --set global.namespace=open-cluster-management \
	--set global.pullPolicy=Always \
	--set global.imageOverrides.cluster_proxy="${IMAGE_CLUSTER_PROXY}:main" \
	--set hubconfig.clusterIngressDomain="${BASEDDOMAIN}"
if [ $? -eq 1 ]; then
  echo "failed to install cluster-proxy addon"
  exit 1
fi

waitForAddon "cluster-proxy" "cluster1"

$KUBECTL wait --for=condition=Available -n cluster1 mca cluster-proxy --timeout=120s
if [ $? -eq 1 ]; then
  echo "cannot wait mca cluster-proxy in cluster1 available"
  $KUBECTL get -n cluster1 mca cluster-proxy -o yaml
  $KUBECTL get -n open-cluster-management pods
  exit 1
fi

cd ../ || exist
rm -rf cluster-proxy-addon

rm -rf managed-serviceaccount
echo "############  Cloning managed-serviceaccount"
git clone --depth 1 --branch "$OCM_BRANCH" https://github.com/stolostron/managed-serviceaccount.git

cd managed-serviceaccount || {
  printf "cd failed, managed-serviceaccount does not exist"
  exit 1
}


../$HELM install \
  -n open-cluster-management --create-namespace \
  managed-serviceaccount charts/managed-serviceaccount/ \
  --set featureGates.ephemeralIdentity=true \
  --set enableAddOnDeploymentConfig=true \
  --set hubDeployMode=AddOnTemplate \
  --set targetCluster=cluster1 \
  --set image=quay.io/stolostron/managed-serviceaccount \
  --set tag=main
if [ $? -eq 1 ]; then
  echo "failed to install managed-serviceaccount addon"
  exit 1
fi

waitForAddon "managed-serviceaccount" "cluster1"

$KUBECTL wait --for=condition=Available -n cluster1 mca managed-serviceaccount --timeout=120s
if [ $? -eq 1 ]; then
  echo "cannot wait mca managed-serviceaccount in cluster1 available"
  $KUBECTL get -n cluster1 mca managed-serviceaccount -o yaml
  exit 1
fi


cd ../ || exist
rm -rf managed-serviceaccount

echo "############  Finished addons installation!!!"
