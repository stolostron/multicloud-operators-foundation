#!/bin/bash

set -o errexi
set -o nounset
set -o pipefail
set -o xtrace

# Run e2e test
export IMAGE_NAME_AND_VERSION=${1}

# Install kubectl
KUBECTL_PATH="${HOME}"/kubectl
mkdir -p "${KUBECTL_PATH}"
wget -P "${KUBECTL_PATH}" https://storage.googleapis.com/kubernetes-release/release/v1.16.2/bin/linux/amd64/kubectl
chmod +x "${KUBECTL_PATH}"/kubectl
export PATH="${KUBECTL_PATH}":"${PATH}"

# Install kind
GO111MODULE="on" go get sigs.k8s.io/kind@v0.7.0

# Create a cluster
cat <<EOF >>kind.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
EOF

CLUSTER_NAME="multicloud-hub"

export KIND_CLUSTER=${CLUSTER_NAME}
kind create cluster --name ${CLUSTER_NAME} --config kind.yaml

# wait cluster is ready
for i in {1..6}; do
  echo "############$i  Checking nodes of cluster"
  READY_NODES=$(kubectl get nodes | grep -c Ready)
  if [ "${READY_NODES}" -eq 2 ]; then
    break
  fi

  if [ $i -eq 6 ]; then
    echo "!!!!!!!!!!  the nodes are not ready in 60s"
    kubectl get nodes
    exit 1
  fi
  sleep 10
done

if ! kubectl cluster-info; then
  echo "Failed to create kind cluster"
  exit 1
fi

IMAGE=$(docker images | grep -c "${IMAGE_NAME_AND_VERSION}")
if [ "${IMAGE}" -ne 1 ]; then
  docker pull "${IMAGE_NAME_AND_VERSION}"
fi

kind load docker-image --name="${CLUSTER_NAME}" --nodes="${CLUSTER_NAME}-worker" "${IMAGE_NAME_AND_VERSION}"

BUILD_PATH=${GOPATH}/src/github.com/open-cluster-management/multicloud-operators-foundation/build

# Deploy ManagedCluster
bash "${BUILD_PATH}"/install-managedcluster.sh
rst="$?"
if [ "$rst" -ne 0 ]; then
  echo "Failed to install managed cluster!!!"
  exit 1
fi

# Deploy acm hub foundation
HUB_PATH=${GOPATH}/src/github.com/open-cluster-management/multicloud-operators-foundation/deploy/dev/hub

cat <<EOF >>"${HUB_PATH}"/kustomization.yaml
resources:
- crds/action.open-cluster-management.io_managedclusteractions.yaml
- crds/internal.open-cluster-management.io_managedclusterinfos.yaml
- crds/inventory.open-cluster-management.io_baremetalassets.yaml
- crds/view.open-cluster-management.io_managedclusterviews.yaml
- 100-clusterrole.yaml
- 100-agent-ca.yaml
- 200-proxyserver.yaml
- 200-controller.yaml

images:
- name: ko://github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-proxyserver
  newName: $IMAGE_NAME_AND_VERSION
- name: ko://github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-controller
  newName: $IMAGE_NAME_AND_VERSION
EOF

kubectl apply -k "${HUB_PATH}"

MANAGED_CLUSTER=$(kubectl get managedclusters | grep cluster | awk '{print $1}')

# Wait for acm foundation hub ready
for i in {1..7}; do
  echo "############$i  Checking acm-controller"
  RUNNING_POD=$(kubectl -n open-cluster-management get pods | grep acm-controller | grep -c "Running")
  if [ "${RUNNING_POD}" -eq 1 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the acm-controller is not ready within 3 minutes"
    kubectl get pods --all-namespaces
    exit 1
  fi
  sleep 30
done

for i in {1..7}; do
  echo "############$i  Checking ManagedClusterInfo"
  INFO=$(kubectl get managedclusterinfos -n "${MANAGED_CLUSTER}" "${MANAGED_CLUSTER}" -o yaml | grep -c "type: ManagedClusterJoined" | tr -d '[:space:]')
  if [ "${INFO}" -eq 1 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "Failed to run e2e test, the acm foundation hub is not ready within 3 minutes"
    ACM_POD=$(kubectl -n open-cluster-management get pods | grep acm-controller | awk '{print $1}')
    kubectl logs -n open-cluster-management "$ACM_POD"
    exit 1
  fi
  sleep 30
done

# Deploy acm foundation agent
WORK_PATH=${GOPATH}/src/github.com/open-cluster-management/multicloud-operators-foundation/deploy/dev/klusterlet/manifestwork
sed -e "s@quay.io/open-cluster-management/multicloud-manager@'$IMAGE_NAME_AND_VERSION'@g" "$WORK_PATH"/agent.yaml | kubectl apply -f -

# Wait for acm foundation agent ready
for i in {1..7}; do
  echo "############$i  Checking acm-agent"
  RUNNING_POD=$(kubectl -n open-cluster-management-agent get pods | grep acm-agent | grep -c "Running")
  if [ "${RUNNING_POD}" -eq 1 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "!!!!!!!!!!  the acm-agent is not ready within 3 minutes"
    kubectl get pods --all-namespaces
    kubectl get manifestwork -n "${MANAGED_CLUSTER}" -o yaml
    exit 1
  fi
  sleep 30
done

for i in {1..7}; do
  echo "############$i  Checking ManagedClusterInfo"
  INFO=$(kubectl get managedclusterinfos -n "${MANAGED_CLUSTER}" "${MANAGED_CLUSTER}" -o yaml | grep -c "version:")
  if [ "${INFO}" -eq 1 ]; then
    break
  fi

  if [ $i -eq 7 ]; then
    echo "Failed to run e2e test, the acm foundation agent is not ready within 3 minutes"
    ACM_POD=$(kubectl -n open-cluster-management-agent get pods | grep acm-agent | awk '{print $1}')
    kubectl logs -n open-cluster-management-agent "$ACM_POD"
    exit 1
  fi
  sleep 30
done

echo "ACM Foundation is deployed successfully!!!"
# Run e2e test
make GO_REQUIRED_MIN_VERSION:= e2e-test

rst="$?"
if [ "$rst" -ne 0 ]; then
  echo "Failed to run e2e-test !!!"
  exit 1
fi

# Test logging
kubectl proxy &
sleep 10
curl http://127.0.0.1:8001/apis/proxy.open-cluster-management.io/v1beta1/namespaces/cluster1/clusterstatuses/cluster1/log/kube-system/etcd-multicloud-hub-control-plane/etcd

rst="$?"
if [ "$rst" -ne 0 ]; then
  echo "Failed to get log !!!"
  exit 1
fi

