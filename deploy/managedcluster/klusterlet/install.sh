#!/bin/bash

set -o errexi
set -o nounset
set -o pipefail

rm -rf registration-operator

echo "############  Cloning registration-operator"
git clone https://github.com/open-cluster-management/registration-operator.git
cd registration-operator || {
  printf "cd failed, registration-operator does not exist"
  return 1
}

echo "############  Deploying klusterlet"
make deploy-spoke

for i in {1..7}; do
  echo "############$i  Checking klusterlet-registration-agent"
  RUNNING_POD=$(kubectl -n open-cluster-management-agent get pods | grep klusterlet-registration-agent | grep -c "Running")
  if [ "${RUNNING_POD}" -eq 3 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the klusterlet-registration-agent is not ready within 3 minutes"
    kubectl -n open-cluster-management-agent get pods
    exit 1
  fi
  sleep 30

done

echo "############  ManagedCluster klusterlet is installed successfully!!"

echo "############  Cleanup"
cd ../ || exist
rm -rf registration-operator

echo "############  Finished installation!!!"
