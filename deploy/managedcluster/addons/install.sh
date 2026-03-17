#!/bin/bash

set -xv
set -o nounset
set -o pipefail

pwd

HELM=${HELM:-_output/tools/bin/helm}

KUBECTL=${KUBECTL:-kubectl}
OCM_BRANCH=${OCM_BRANCH:-backplane-2.10}

CLUSTER_PROXY_ADDON_IMAGE=${CLUSTER_PROXY_ADDON_IMAGE:-quay.io/stolostron/cluster-proxy-addon}
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


rm -rf cluster-proxy-addon
echo "############  Cloning cluster-proxy-addon"
git clone --depth 1 --branch "$OCM_BRANCH" https://github.com/stolostron/cluster-proxy-addon.git

cd cluster-proxy-addon || {
  printf "cd failed, cluster-proxy-addon does not exist"
  exit 1
}

BASEDDOMAIN=$($KUBECTL get ingress.config.openshift.io cluster -o=jsonpath='{.spec.domain}')

../$HELM install \
	-n open-cluster-management --create-namespace \
	cluster-proxy-addon chart/cluster-proxy-addon \
	--set global.pullPolicy=Always \
	--set global.imageOverrides.cluster_proxy_addon="${CLUSTER_PROXY_ADDON_IMAGE}:backplane-2.10" \
	--set global.imageOverrides.cluster_proxy="${IMAGE_CLUSTER_PROXY}:backplane-2.10" \
	--set cluster_basedomain="${BASEDDOMAIN}"
if [ $? -eq 1 ]; then
  echo "failed to install cluster-proxy addon"
  exit 1
fi

# cluster-proxy-addon takes a long time to become available, wait 8 minutes before checking
sleep 480
waitForAddon "cluster-proxy" "cluster1"

$KUBECTL wait --for=condition=Available -n cluster1 mca cluster-proxy --timeout=120s
if [ $? -eq 1 ]; then
  echo "cannot wait mca cluster-proxy in cluster1 available"
  $KUBECTL get -n cluster1 mca cluster-proxy -o yaml
  $KUBECTL get -n open-cluster-management pods
  exit 1
fi

# Print last 50 lines of proxy-server logs to verify agent connectivity to hub
PROXY_SERVER_POD=$($KUBECTL get pods -n open-cluster-management -l proxy.open-cluster-management.io/component-name=proxy-server -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
if [ -n "$PROXY_SERVER_POD" ]; then
  # Use the first pod if multiple are found
  PROXY_SERVER_POD=$(echo "$PROXY_SERVER_POD" | awk '{print $1}')
  echo "############  Proxy-server pod logs (last 50 lines):"
  $KUBECTL logs -n open-cluster-management "$PROXY_SERVER_POD" --tail=50
else
  echo "WARNING: proxy-server pod not found in open-cluster-management namespace"
  $KUBECTL get pods -n open-cluster-management
fi

# Print proxy-agent pod status and logs on the managed cluster side
echo "############  Proxy-agent pods in open-cluster-management-agent-addon namespace:"
$KUBECTL get pods -n open-cluster-management-agent-addon -l open-cluster-management.io/addon-name=cluster-proxy -o wide
PROXY_AGENT_PODS=$($KUBECTL get pods -n open-cluster-management-agent-addon -l open-cluster-management.io/addon-name=cluster-proxy -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
if [ -n "$PROXY_AGENT_PODS" ]; then
  for pod in $PROXY_AGENT_PODS; do
    echo "############  Proxy-agent pod logs ($pod, last 50 lines):"
    $KUBECTL logs -n open-cluster-management-agent-addon "$pod" --tail=50 --all-containers
  done
else
  echo "WARNING: No proxy-agent pods found in open-cluster-management-agent-addon namespace"
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
  --set tag="${OCM_BRANCH}"
if [ $? -eq 1 ]; then
  echo "failed to install managed-serviceaccount addon"
  exit 1
fi

# managed-serviceaccount takes a long time to become available, wait 5 minutes before checking
sleep 300
waitForAddon "managed-serviceaccount" "cluster1"

$KUBECTL wait --for=condition=Available -n cluster1 mca managed-serviceaccount --timeout=120s
if [ $? -eq 1 ]; then
  echo "cannot wait mca managed-serviceaccount in cluster1 available"

  echo "############  MCA status:"
  $KUBECTL get -n cluster1 mca managed-serviceaccount -o yaml

  echo "############  ManifestWorks for managed-serviceaccount in cluster1:"
  $KUBECTL get manifestwork -n cluster1 | grep managed-serviceaccount
  $KUBECTL get manifestwork -n cluster1 -l open-cluster-management.io/addon-name=managed-serviceaccount -o yaml

  echo "############  Addon agent deployment and pods in open-cluster-management-agent-addon namespace:"
  $KUBECTL get deploy,pods -n open-cluster-management-agent-addon | grep managed-serviceaccount
  $KUBECTL get pods -n open-cluster-management-agent-addon -l open-cluster-management.io/addon-name=managed-serviceaccount -o wide

  echo "############  Addon agent pod details (describe + logs):"
  MSA_PODS=$($KUBECTL get pods -n open-cluster-management-agent-addon -l open-cluster-management.io/addon-name=managed-serviceaccount -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
  if [ -n "$MSA_PODS" ]; then
    for pod in $MSA_PODS; do
      echo "--- Describing pod: $pod ---"
      $KUBECTL describe pod -n open-cluster-management-agent-addon "$pod"
      echo "--- Logs from pod: $pod ---"
      $KUBECTL logs -n open-cluster-management-agent-addon "$pod" --tail=80 --all-containers
    done
  else
    echo "WARNING: No managed-serviceaccount agent pods found"
  fi

  echo "############  Events in open-cluster-management-agent-addon namespace:"
  $KUBECTL get events -n open-cluster-management-agent-addon --sort-by='.lastTimestamp' | tail -30

  echo "############  Addon manager pods and logs on hub:"
  $KUBECTL get pods -n open-cluster-management | grep managed-serviceaccount
  MSA_MGR_POD=$($KUBECTL get pods -n open-cluster-management -l app=managed-serviceaccount-addon-manager -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
  if [ -n "$MSA_MGR_POD" ]; then
    echo "--- Addon manager logs (last 80 lines): ---"
    $KUBECTL logs -n open-cluster-management "$MSA_MGR_POD" --tail=80
  fi

  exit 1
fi

cd ../ || exist
rm -rf managed-serviceaccount

echo "############  Finished addons installation!!!"
