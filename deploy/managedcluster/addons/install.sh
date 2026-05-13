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
oc apply -f https://raw.githubusercontent.com/stolostron/cluster-proxy/$OCM_BRANCH/charts/cluster-proxy/crds/managedproxyconfigurations.yaml

../$HELM install \
	-n open-cluster-management --create-namespace \
	cluster-proxy-addon pkg/templates/charts/toggle/cluster-proxy-addon \
  --set global.namespace=open-cluster-management \
	--set global.pullPolicy=Always \
	--set global.imageOverrides.cluster_proxy="${IMAGE_CLUSTER_PROXY}:${OCM_BRANCH}" \
	--set hubconfig.clusterIngressDomain="${BASEDDOMAIN}"
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
fi

# Always print full cluster-proxy diagnostic regardless of success/failure
echo ""
echo "============================================================"
echo "  CLUSTER-PROXY ADDON DIAGNOSTIC REPORT"
echo "============================================================"

# 1. ManagedClusterAddOn status and conditions
echo ""
echo "############  [1/8] ManagedClusterAddOn status:"
$KUBECTL get -n cluster1 mca cluster-proxy -o yaml

# 2. ManifestWork status (the bridge between hub and managed cluster)
echo ""
echo "############  [2/8] ManifestWorks for cluster-proxy:"
$KUBECTL get manifestwork -n cluster1 | grep -E "NAME|cluster-proxy"
PROXY_MWS=$($KUBECTL get manifestwork -n cluster1 -l open-cluster-management.io/addon-name=cluster-proxy -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
if [ -n "$PROXY_MWS" ]; then
  for mw in $PROXY_MWS; do
    echo "--- ManifestWork: $mw ---"
    echo "  Conditions:"
    $KUBECTL get manifestwork -n cluster1 "$mw" -o jsonpath='{range .status.conditions[*]}  - {.type}: {.status} ({.reason}) {.message}{"\n"}{end}'
    echo "  Resource Status:"
    $KUBECTL get manifestwork -n cluster1 "$mw" -o jsonpath='{range .status.resourceStatus.manifests[*]}  - {.resourceMeta.group}/{.resourceMeta.resource} {.resourceMeta.namespace}/{.resourceMeta.name}: {range .conditions[*]}{.type}={.status} {end}{"\n"}{end}'
  done
else
  echo "WARNING: No ManifestWorks found for cluster-proxy addon"
fi

# 3. Addon manager pod (hub side)
echo ""
echo "############  [3/8] Cluster-proxy addon-manager pod (hub):"
$KUBECTL get pods -n open-cluster-management -l component=cluster-proxy-addon-manager -o wide
ADDON_MGR_POD=$($KUBECTL get pods -n open-cluster-management -l component=cluster-proxy-addon-manager -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$ADDON_MGR_POD" ]; then
  echo "--- Addon manager logs (last 80 lines): ---"
  $KUBECTL logs -n open-cluster-management "$ADDON_MGR_POD" --tail=80
fi

# 4. Proxy-server pod (hub side)
echo ""
echo "############  [4/8] Proxy-server pod (hub):"
PROXY_SERVER_POD=$($KUBECTL get pods -n open-cluster-management -l proxy.open-cluster-management.io/component-name=proxy-server -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
if [ -n "$PROXY_SERVER_POD" ]; then
  PROXY_SERVER_POD=$(echo "$PROXY_SERVER_POD" | awk '{print $1}')
  $KUBECTL get pod -n open-cluster-management "$PROXY_SERVER_POD" -o wide
  echo "--- Proxy-server logs (last 80 lines): ---"
  $KUBECTL logs -n open-cluster-management "$PROXY_SERVER_POD" --tail=80
else
  echo "WARNING: proxy-server pod not found"
  $KUBECTL get pods -n open-cluster-management
fi

# 5. Proxy-agent deployment and pods (managed cluster side)
echo ""
echo "############  [5/8] Proxy-agent deployment and pods (managed cluster):"
$KUBECTL get deploy -n open-cluster-management-agent-addon -l open-cluster-management.io/addon-name=cluster-proxy -o wide
$KUBECTL get pods -n open-cluster-management-agent-addon -l open-cluster-management.io/addon-name=cluster-proxy -o wide
PROXY_AGENT_PODS=$($KUBECTL get pods -n open-cluster-management-agent-addon -l open-cluster-management.io/addon-name=cluster-proxy -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
if [ -n "$PROXY_AGENT_PODS" ]; then
  for pod in $PROXY_AGENT_PODS; do
    echo "--- Describing pod: $pod ---"
    $KUBECTL describe pod -n open-cluster-management-agent-addon "$pod"
    echo "--- Logs from pod: $pod (last 80 lines, all containers): ---"
    $KUBECTL logs -n open-cluster-management-agent-addon "$pod" --tail=80 --all-containers
    echo "--- Previous logs (if any): ---"
    $KUBECTL logs -n open-cluster-management-agent-addon "$pod" --tail=40 --all-containers --previous 2>/dev/null || echo "  (no previous logs)"
  done
else
  echo "WARNING: No proxy-agent pods found in open-cluster-management-agent-addon namespace"
  echo "--- All pods in open-cluster-management-agent-addon: ---"
  $KUBECTL get pods -n open-cluster-management-agent-addon -o wide
  echo "--- All deployments in open-cluster-management-agent-addon: ---"
  $KUBECTL get deploy -n open-cluster-management-agent-addon -o wide
fi

# 6. Events (look for eviction, OOMKilled, scheduling failures)
echo ""
echo "############  [6/8] Recent events in open-cluster-management-agent-addon (last 30):"
$KUBECTL get events -n open-cluster-management-agent-addon --sort-by='.lastTimestamp' 2>/dev/null | tail -30

# 7. Node resource pressure (check if pod was evicted due to resource pressure)
echo ""
echo "############  [7/8] Node resource status:"
$KUBECTL get nodes -o wide
echo "--- Node conditions (MemoryPressure, DiskPressure, PIDPressure): ---"
$KUBECTL get nodes -o jsonpath='{range .items[*]}Node: {.metadata.name}{"\n"}{range .status.conditions[*]}  {.type}: {.status} (reason={.reason}){"\n"}{end}{end}'
echo "--- Node resource usage: ---"
$KUBECTL top nodes 2>/dev/null || echo "  (metrics-server not available)"

# 8. Work-agent status (responsible for applying ManifestWork)
echo ""
echo "############  [8/8] Work-agent status (klusterlet):"
$KUBECTL get pods -n open-cluster-management-agent -o wide | grep -E "NAME|work"
WORK_AGENT_POD=$($KUBECTL get pods -n open-cluster-management-agent -l app=klusterlet-manifestwork-agent -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$WORK_AGENT_POD" ]; then
  echo "--- Work-agent logs (last 40 lines): ---"
  $KUBECTL logs -n open-cluster-management-agent "$WORK_AGENT_POD" --tail=40
fi

echo ""
echo "============================================================"
echo "  END OF CLUSTER-PROXY DIAGNOSTIC REPORT"
echo "============================================================"
echo ""

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

echo ""
echo "============================================================"
echo "  PRE-TEST ADDON STATUS SNAPSHOT"
echo "============================================================"
echo ""
echo "############  All ManagedClusterAddOns in cluster1:"
$KUBECTL get mca -n cluster1
echo ""
echo "############  All pods in open-cluster-management-agent-addon:"
$KUBECTL get pods -n open-cluster-management-agent-addon -o wide
echo ""
echo "############  All pods in open-cluster-management (hub):"
$KUBECTL get pods -n open-cluster-management -o wide
echo ""
echo "############  Cluster-proxy agent pod status (detailed):"
PROXY_AGENT_PODS=$($KUBECTL get pods -n open-cluster-management-agent-addon -l open-cluster-management.io/addon-name=cluster-proxy -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
if [ -n "$PROXY_AGENT_PODS" ]; then
  for pod in $PROXY_AGENT_PODS; do
    echo "--- Container statuses for pod: $pod ---"
    $KUBECTL get pod -n open-cluster-management-agent-addon "$pod" -o jsonpath='{range .status.containerStatuses[*]}  Container: {.name}, Ready: {.ready}, RestartCount: {.restartCount}, State: {.state}{"\n"}{end}'
    echo "--- Last terminated status (if restarted): ---"
    $KUBECTL get pod -n open-cluster-management-agent-addon "$pod" -o jsonpath='{range .status.containerStatuses[*]}  Container: {.name}, LastTermination: {.lastState.terminated.reason} (exit={.lastState.terminated.exitCode}){"\n"}{end}'
  done
else
  echo "WARNING: proxy-agent pod STILL not found before test execution"
  echo "--- Checking if deployment exists: ---"
  $KUBECTL get deploy -n open-cluster-management-agent-addon -o wide
  echo "--- Recent events: ---"
  $KUBECTL get events -n open-cluster-management-agent-addon --sort-by='.lastTimestamp' 2>/dev/null | tail -20
  echo "--- Node conditions: ---"
  $KUBECTL get nodes -o jsonpath='{range .items[*]}Node: {.metadata.name}{"\n"}{range .status.conditions[*]}  {.type}: {.status}{"\n"}{end}{end}'
fi
echo ""
echo "============================================================"
echo "  END PRE-TEST SNAPSHOT"
echo "============================================================"

echo "############  Finished addons installation!!!"
