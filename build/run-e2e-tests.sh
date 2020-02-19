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
CLUSTER_NAME="multicloud-hub"
kind create cluster --name ${CLUSTER_NAME}

if ! kubectl cluster-info
then
    echo "Failed to create kind cluster"
    exit 1
fi

kind load docker-image --name="${CLUSTER_NAME}" "${IMAGE_NAME_AND_VERSION}"

# Deploy hub cluster
HUB_PATH=${GOPATH}/src/github.com/open-cluster-management/multicloud-operators-foundation/deploy/dev/hub

cat <<EOF >>"${HUB_PATH}"/kustomization.yaml
images:
- name: github.com/open-cluster-management/multicloud-operators-foundation/cmd/mcm-apiserver
  newName: $IMAGE_NAME_AND_VERSION
- name: github.com/open-cluster-management/multicloud-operators-foundation/cmd/mcm-controller
  newName: $IMAGE_NAME_AND_VERSION
EOF

kubectl apply -k "${HUB_PATH}"

# Wait for the hub cluster ready
for i in {1..7}; do
    if ! kubectl get clusters --all-namespaces
    then
        if [ $i -eq 7 ]; then
            echo "Failed to run e2e test, the hub cluster is not ready within 3 minutes"
            kubectl -n multicloud-system get pods
            exit 1
        fi
        sleep 30
    else
        break
    fi
done

# Run e2e test
make e2e-test
