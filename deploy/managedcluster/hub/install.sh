#!/bin/bash

set -o nounset
set -o pipefail

KUBECTL=${KUBECTL:-kubectl}
REGISTRATION_OPERATOR_BRANCH=${REGISTRATION_OPERATOR_BRANCH:-release-2.3}

# On openshift, OLM is installed into openshift-operator-lifecycle-manager
$KUBECTL get namespace openshift-operator-lifecycle-manager 1>/dev/null 2>&1
if [ $? -eq 0 ]; then
  export OLM_NAMESPACE=openshift-operator-lifecycle-manager
fi

rm -rf registration-operator

echo "############  Cloning registration-operator"
git clone --depth 1 --branch "$REGISTRATION_OPERATOR_BRANCH" https://github.com/stolostron/registration-operator.git

cd registration-operator || {
  printf "cd failed, registration-operator does not exist"
  return 1
}

echo "############  Deploying hub"
make deploy-hub
if [ $? -ne 0 ]; then
 echo "############  Failed to deploy hub"
 exit 1
fi

for i in {1..7}; do
  echo "############$i  Checking cluster-manager-registration-controller"
  RUNNING_POD=$($KUBECTL -n open-cluster-management-hub get pods | grep cluster-manager-registration-controller | grep -c "Running")
  if [ "${RUNNING_POD}" -ge 1 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the cluster-manager-registration-controller is not ready within 3 minutes"
    $KUBECTL -n open-cluster-management-hub get pods

    exit 1
  fi
  sleep 30
done

for i in {1..7}; do
  echo "############$i  Checking cluster-manager-registration-webhook"
  RUNNING_POD=$($KUBECTL -n open-cluster-management-hub get pods | grep cluster-manager-registration-webhook | grep -c "Running")
  if [ "${RUNNING_POD}" -ge 1 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the cluster-manager-registration-webhook is not ready within 3 minutes"
    $KUBECTL -n open-cluster-management-hub get pods
    exit 1
  fi
  sleep 30
done

echo "############  ManagedCluster hub is installed successfully!!"

echo "############  Cleanup"
cd ../ || exist
rm -rf registration-operator

echo "############  Finished installation!!!"
