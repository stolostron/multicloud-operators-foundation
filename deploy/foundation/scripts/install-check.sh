#!/bin/bash

set -o nounset
set -o pipefail

KUBECTL=${KUBECTL:-kubectl}


for i in {1..7}; do
  echo "############$i  Checking foundation pods"
  RUNNING_POD=0
  controller=$($KUBECTL -n open-cluster-management get pods | grep foundation-controller | grep -c "Running")
  RUNNING_POD=$((RUNNING_POD+controller))
  proxyserver=$($KUBECTL -n open-cluster-management get pods | grep foundation-proxyserver | grep -c "Running")
  RUNNING_POD=$((RUNNING_POD+proxyserver))
  webhook=$($KUBECTL -n open-cluster-management get pods | grep foundation-webhook | grep -c "Running")
  RUNNING_POD=$((RUNNING_POD+webhook))
  agent=$($KUBECTL -n open-cluster-management-agent get pods | grep foundation-agent | grep -c "Running")
  RUNNING_POD=$((RUNNING_POD+agent))

  if [ "${RUNNING_POD}" -eq 4 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the foundation pods are not ready within 4 minutes"
    $KUBECTL -n open-cluster-management get pods
    $KUBECTL -n open-cluster-management get secret
    $KUBECTL -n open-cluster-management-agent get pods
    $KUBECTL get mcl
    $KUBECTL -n cluster1 get manifestworks.work.open-cluster-management.io -o yaml
    exit 1
  fi
  sleep 30
done


echo "############  Foundation is installed successfully!!"

