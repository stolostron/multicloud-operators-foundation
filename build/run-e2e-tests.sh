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
CLUSTER_NAME="kind"
kind create cluster --name ${CLUSTER_NAME}

if ! kubectl cluster-info
then
    echo "Failed to create kind cluster"
    exit 1
fi

kind load docker-image --name="${CLUSTER_NAME}" "${IMAGE_NAME_AND_VERSION}"

#deploy registration-operator
git clone https://github.com/open-cluster-management/registration-operator.git
cd registration-operator || { printf "cd failed, registration-operator do not exist"; return 1; }

#deploy hub core
make GO_REQUIRED_MIN_VERSION:= deploy-hub

for i in {1..7}; do
    RUNNING_HUB=$(kubectl -n open-cluster-management get pods |grep cluster-manager| grep -c Running)
    if [ "${RUNNING_HUB}" -eq 3 ]
    then
        break
    else
        if [ $i -eq 7 ]; then
            echo "Failed to run e2e test, the hub cluster is not ready within 3 minutes"
            kubectl -n open-cluster-management get pods 
            exit 1
        fi
        sleep 30
    fi
done

for i in {1..7}; do
    RUNNING_HUB_CONTROLLER=$(kubectl -n open-cluster-management-hub get pods |grep cluster-manager-registration| grep -c "Running")
    if [ "${RUNNING_HUB_CONTROLLER}" -eq 6 ]
    then
        break
    else
        if [ $i -eq 7 ]; then
            echo "Failed to run e2e test, the hub controller is not ready within 3 minutes"
            kubectl -n open-cluster-management-hub get pods
            exit 1
        fi
        sleep 30
    fi
done

#deploy spokecluster
cp ~/.kube/config ./.kubeconfig
make GO_REQUIRED_MIN_VERSION:= deploy-spoke

for i in {1..7}; do
    RUNNING_SPOKE=$(kubectl -n open-cluster-management get pods |grep klusterlet| grep -c "Running")
    if [ "${RUNNING_SPOKE}" -eq 3 ]
    then
        break
    else
        if [ $i -eq 7 ]; then
            echo "Failed to run e2e test, the spoke operator is not ready within 3 minutes"
            kubectl -n open-cluster-management get pods

            exit 1
        fi
        sleep 30
    fi
done

for i in {1..7}; do
    kubectl get pods --all-namespaces 
    RUNNING_SPOKE_AGENT=$(kubectl -n open-cluster-management-spoke get pods |grep klusterlet| grep -c "Running")
    if [ "${RUNNING_SPOKE_AGENT}" -eq 6 ]
    then
        break
    else
        if [ $i -eq 7 ]; then
            echo "Failed to run e2e test, the spoke registration-agent is not ready within 3 minutes"
            kubectl -n open-cluster-management-spoke get pods 
            exit 1
        fi
        sleep 30
    fi
done

#approve csr
for i in {1..7}; do
    if ! kubectl get csr 
    then
        if [ $i -eq 7 ]; then
            echo "Failed to run e2e test, the hub cluster is not ready within 3 minutes"
            kubectl get csr 
            exit 1
        fi
        sleep 30
    else
        CSR_NAME=$(kubectl get csr |grep cluster |awk '{print $1}')
        kubectl certificate approve "${CSR_NAME}"
        break
    fi
done


for i in {1..7}; do
    SPOKE_CLUSTER_NUM=$(kubectl get spokeclusters | grep -c cluster |tr -d '[:space:]')
    if [ "${SPOKE_CLUSTER_NUM}" -eq 1 ]
    then
        break
    else
        if [ $i -eq 7 ]; then
            echo "Failed to run e2e test, the spoke cluster is not created within 3 minutes"
            kubectl get spokeclusters
            exit 1
        fi
        sleep 30
    fi
done

kubectl get spokeclusters -oyaml > spokecluster.yaml

sed -i "s/hubAcceptsClient: false/hubAcceptsClient: true/g" spokecluster.yaml

kubectl apply -f spokecluster.yaml

for i in {1..7}; do
    SPOKE_CLUSTER_ACCEPT=$(kubectl get spokeclusters -oyaml | grep -c "type: SpokeClusterJoined"|tr -d '[:space:]')
    if [ "${SPOKE_CLUSTER_ACCEPT}" -eq 1 ]
    then
        break
    else
        if [ $i -eq 7 ]; then
            echo "Failed to run e2e test, the spoke cluster is not created within 3 minutes"
            kubectl get spokeclusters -oyaml
            exit 1
        fi
        sleep 30
    fi
done

kubectl get spokeclusters -oyaml


# Deploy hub cluster
HUB_PATH=${GOPATH}/src/github.com/open-cluster-management/multicloud-operators-foundation/deploy/dev/hub

cat <<EOF >>"${HUB_PATH}"/kustomization.yaml
images:
- name: github.com/open-cluster-management/multicloud-operators-foundation/cmd/mcm-apiserver
  newName: $IMAGE_NAME_AND_VERSION
- name: github.com/open-cluster-management/multicloud-operators-foundation/cmd/mcm-controller
  newName: $IMAGE_NAME_AND_VERSION
- name: github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-proxyserver
  newName: $IMAGE_NAME_AND_VERSION
- name: github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-controller
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

cd ../ || return 1;
# Run e2e test
make GO_REQUIRED_MIN_VERSION:= e2e-test
