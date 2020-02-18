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
kind create cluster

# Check kind cluster
if ! kubectl cluster-info
then
    echo "Failed to create kind cluster"
    exit 1
fi

# Deploy hub cluster
kubectl create namespace multicloud-system

kubectl create secret docker-registry -n multicloud-system mcm-image-pull-secret \
  --docker-server=quay.io \
  --docker-username="${DOCKER_USER}" \
  --docker-password="${DOCKER_PASS}"

HUB_PATH=${GOPATH}/src/github.com/open-cluster-management/multicloud-operators-foundation/deploy/dev/hub

cat <<EOF >>"${HUB_PATH}"/serviceaccount-patch.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hub-sa
  namespace: multicloud-system
imagePullSecrets:
- name: mcm-image-pull-secret
EOF

cat <<EOF >>"${HUB_PATH}"/kustomization.yaml
images:
- name: github.com/open-cluster-management/multicloud-operators-foundation/cmd/mcm-apiserver
  newName: $IMAGE_NAME_AND_VERSION
- name: github.com/open-cluster-management/multicloud-operators-foundation/cmd/mcm-controller
  newName: $IMAGE_NAME_AND_VERSION
patchesStrategicMerge:
- serviceaccount-patch.yaml
EOF

kubectl apply -k "${HUB_PATH}"

# Wait for the hub cluster ready
for i in {1..7}; do
    if [ $i -eq 7 ]; then
        echo "Failed to run e2e test, the hub cluster is not ready..."
        kubectl -n multicloud-system get pods
        kubectl get pods -n multicloud-system | grep mcm-apiserver | awk '{print $1}' | xargs kubectl -n multicloud-system describe pod
        kubectl get pods -n multicloud-system | grep mcm-apiserver | awk '{print $1}' | xargs kubectl -n multicloud-system logs -f
        exit 1
    fi

    if ! kubectl get clusters --all-namespaces
    then
        sleep 30
    else
        break
    fi
done
