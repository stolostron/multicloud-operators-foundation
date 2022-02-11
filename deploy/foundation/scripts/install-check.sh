#!/bin/bash

set -xv
set -o nounset
set -o pipefail

KUBECTL=${KUBECTL:-kubectl}


for i in {1..7}; do
  echo "############$i  Checking foundation pods"
  RUNNING_POD=0
  controller=$($KUBECTL -n open-cluster-management get pods | grep ocm-controller | grep -c "Running")
  RUNNING_POD=$((RUNNING_POD+controller))
  proxyserver=$($KUBECTL -n open-cluster-management get pods | grep ocm-proxyserver | grep -c "Running")
  RUNNING_POD=$((RUNNING_POD+proxyserver))
  webhook=$($KUBECTL -n open-cluster-management get pods | grep ocm-webhook | grep -c "Running")
  RUNNING_POD=$((RUNNING_POD+webhook))
  agent=$($KUBECTL -n open-cluster-management-agent-addon get pods | grep klusterlet-addon-workmgr | grep -c "Running")
  RUNNING_POD=$((RUNNING_POD+agent))

  if [ "${RUNNING_POD}" -eq 4 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the foundation pods are not ready within 4 minutes"
    $KUBECTL -n open-cluster-management get pods -o yaml
    $KUBECTL -n open-cluster-management-hub get pods -o yaml
    $KUBECTL -n open-cluster-management-agent get pods -o yaml
    $KUBECTL -n open-cluster-management-agent-addon get pods -o yaml
    $KUBECTL get klusterlet klusterlet -o yaml
    $KUBECTL get mcl cluster1 -o yaml
    $KUBECTL -n cluster1 get manifestworks.work.open-cluster-management.io -o yaml
    $KUBECTL get pods -n open-cluster-management -l app=ocm-controller | grep ocm-controller | awk '{print $1}' |xargs $KUBECTL -n open-cluster-management logs
    $KUBECTL get pods -n open-cluster-management -l app=ocm-proxyserver | grep ocm-proxyserver | awk '{print $1}' |xargs $KUBECTL -n open-cluster-management logs

    exit 1
  fi
  sleep 30
done


echo "############  Foundation is installed successfully!!"

