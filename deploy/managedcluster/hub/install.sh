#!/bin/bash

echo "############  Cloning registration-operator"
git clone https://github.com/open-cluster-management/registration-operator.git
cd registration-operator || {
  printf "cd failed, registration-operator does not exist"
  return 1
}

echo "############  Deploying hub"
make deploy-hub

for i in {1..7}; do
  echo "############$i  Checking cluster-manager-registration-controller"
  RUNNING_POD=$(kubectl -n open-cluster-management-hub get pods | grep cluster-manager-registration-controller | grep -c "Running")
  if [ "${RUNNING_POD}" -eq 3 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the cluster-manager-registration-controller is not ready within 3 minutes"
    kubectl -n open-cluster-management-hub get pods

    exit 1
  fi
  sleep 30
done

for i in {1..7}; do
  echo "############$i  Checking cluster-manager-registration-webhook"
  RUNNING_POD=$(kubectl -n open-cluster-management-hub get pods | grep cluster-manager-registration-webhook | grep -c "Running")
  if [ "${RUNNING_POD}" -eq 3 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the cluster-manager-registration-webhook is not ready within 3 minutes"
    kubectl -n open-cluster-management-hub get pods
    exit 1
  fi
  sleep 30
done

echo "############  ManagedCluster hub is installed successfully!!"

echo "############  Cleanup"
cd ../ || exist
rm -rf registration-operator

echo "############  Finished installation!!!"
