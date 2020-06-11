#!/bin/bash

echo "############  Cloning registration-operator"
git clone https://github.com/open-cluster-management/registration-operator.git
cd registration-operator || {
  printf "cd failed, registration-operator does not exist"
  return 1
}
cp ~/.kube/config ./.kubeconfig

echo "############  Deploying all"
make GO_REQUIRED_MIN_VERSION:= deploy

for i in {1..7}; do
  echo "############$i  Checking cluster-manager-registration-controller"
  RUNNING_POD=$(kubectl -n open-cluster-management-hub get pods | grep cluster-manager-registration-controller | grep -c "Running")
  if [ "${RUNNING_POD}" -eq 3 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the cluster-manager-registration-controller is not ready within 3 minutes"
    kubectl get pods --all-namespaces

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
    kubectl get pods --all-namespaces
    exit 1
  fi
  sleep 30
done

for i in {1..7}; do
  echo "############$i  Checking klusterlet-registration-agent"
  RUNNING_POD=$(kubectl -n open-cluster-management-agent get pods | grep klusterlet-registration-agent | grep -c "Running")
  if [ "${RUNNING_POD}" -eq 3 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the klusterlet-registration-agent is not ready within 3 minutes"
    kubectl get pods --all-namespaces
    exit 1
  fi
  sleep 30
done

for i in {1..7}; do
  echo "############$i  Checking klusterlet-work-agent"
  RUNNING_POD=$(kubectl -n open-cluster-management-agent get pods | grep klusterlet-work-agent | grep -c "Running")
  if [ "${RUNNING_POD}" -eq 3 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the klusterlet-work-agent is not ready within 3 minutes"
    kubectl get pods --all-namespaces
    exit 1
  fi
  sleep 30
done

echo "############  Cluster Manager is installed successfully!!"

for i in {1..7}; do
  echo "############$i  Checking ManagedCluster"
  MANAGED_CLUSTER_NUM=$(kubectl get managedclusters | grep -c cluster | tr -d '[:space:]')
  if [ "${MANAGED_CLUSTER_NUM}" -eq 1 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the managed cluster is not created within 3 minutes"
    kubectl get pods --all-namespaces
    exit 1
  fi
  sleep 30
done

MANAGED_CLUSTER=$(kubectl get managedclusters | grep cluster | awk '{print $1}')

echo "############  managedcluster ${MANAGED_CLUSTER} is created "

for i in {1..7}; do
  echo "############$i  Approve csr"
  CSR_NAME=$(kubectl get csr | grep "${MANAGED_CLUSTER}" | grep Pending | awk '{print $1}')
  if [ -n "${CSR_NAME}" ]; then
    kubectl certificate approve "${CSR_NAME}"
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the csr is not created within 3 minutes"
    kubectl get csr
    exit 1
  fi
  sleep 30
done

echo "############  Accept managedcluster ${MANAGED_CLUSTER}"
kubectl patch managedclusters "${MANAGED_CLUSTER}" --type merge --patch '{"spec":{"hubAcceptsClient":true}}'

for i in {1..7}; do
  echo "############$i  Checking managedcluster ${MANAGED_CLUSTER} joining"
  MANAGED_CLUSTER_ACCEPT=$(kubectl get managedclusters "${MANAGED_CLUSTER}" -o yaml | grep -c "type: ManagedClusterJoined" | tr -d '[:space:]')
  if [ "${MANAGED_CLUSTER_ACCEPT}" -eq 1 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the managed cluster is not joined within 3 minutes"
    kubectl get managedclusters "${MANAGED_CLUSTER}" -o yaml
    exit 1
  fi
  sleep 30
done

echo "############  managedcluster ${MANAGED_CLUSTER} is joined"

echo "############  Cleanup"
cd ../ || exist
rm -rf registration-operator

echo "############  Finished installation!!!"
