#!/bin/bash

set -o nounset
set -o pipefail

KUBECTL=${KUBECTL:-kubectl}

KLUSTERLET_KUBECONFIG_CONTEXT=$($KUBECTL config current-context)
KIND_CLUSTER=kind

# On openshift, OLM is installed into openshift-operator-lifecycle-manager
$KUBECTL get namespace openshift-operator-lifecycle-manager 1>/dev/null 2>&1
if [ $? -eq 0 ]; then
  export OLM_NAMESPACE=openshift-operator-lifecycle-manager
fi

rm -rf registration-operator

echo "############  Cloning registration-operator"
git clone https://github.com/open-cluster-management/registration-operator.git

cd registration-operator || {
  printf "cd failed, registration-operator does not exist"
  return 1
}

echo "############  Deploying klusterlet"
#make deploy-spoke
#if [ $? -ne 0 ]; then
#  echo "############  Failed to deploy klusterlet"
#  exit 1
#fi


CLUSTER_IP=$($KUBECTL get svc kubernetes -n default -o jsonpath="{.spec.clusterIP}")
cp ${KUBECONFIG} dev-kubeconfig
$KUBECTL config use-context ${KLUSTERLET_KUBECONFIG_CONTEXT}

ns=$($KUBECTL get ns open-cluster-management-agent | grep -c "open-cluster-management-agent")
if [ "${ns}" -eq 0 ]; then
  $KUBECTL create ns open-cluster-management-agent
fi

$KUBECTL config set clusters.kind-${KIND_CLUSTER}.server https://${CLUSTER_IP} --kubeconfig dev-kubeconfig
$KUBECTL delete secret bootstrap-hub-kubeconfig -n open-cluster-management-agent --ignore-not-found
$KUBECTL create secret generic bootstrap-hub-kubeconfig --from-file=kubeconfig=dev-kubeconfig -n open-cluster-management-agent

$KUBECTL apply -f deploy/klusterlet/crds/0000_00_operator.open-cluster-management.io_klusterlets.crd.yaml
$KUBECTL apply -f deploy/klusterlet/cluster_role.yaml
$KUBECTL apply -f deploy/klusterlet/cluster_role_binding.yaml
$KUBECTL apply -f deploy/klusterlet/service_account.yaml
$KUBECTL apply -f deploy/klusterlet/operator.yaml
$KUBECTL apply -f deploy/klusterlet/crds/operator_open-cluster-management_klusterlets.cr.yaml

for i in {1..7}; do
  echo "############$i  Checking klusterlet-registration-agent"
  RUNNING_POD=$($KUBECTL -n open-cluster-management-agent get pods | grep klusterlet-registration-agent | grep -c "Running")
  if [ "${RUNNING_POD}" -eq 3 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the klusterlet-registration-agent is not ready within 3 minutes"
    $KUBECTL -n open-cluster-management-agent get pods
    exit 1
  fi
  sleep 30

done

echo "############  ManagedCluster klusterlet is installed successfully!!"

echo "############  Cleanup"
cd ../ || exist
rm -rf registration-operator

echo "############  Finished installation!!!"
