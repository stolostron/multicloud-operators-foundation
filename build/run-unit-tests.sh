#!/bin/bash

# Run unit test
echo "UNIT TESTS GO HERE!"

echo "Install Kubebuilder components for test framework usage!"

_OS=$(go env GOOS)
_ARCH=$(go env GOARCH)

# download kubebuilder and extract it to tmp
curl -L https://go.kubebuilder.io/dl/2.2.0/"${_OS}"/"${_ARCH}" | tar -xz -C /tmp/

# move to a long-term location and put it on your path
# (you'll need to set the KUBEBUILDER_ASSETS env var if you put it somewhere else)
sudo mv /tmp/kubebuilder_2.2.0_"${_OS}"_"${_ARCH}" /usr/local/kubebuilder
export PATH=$PATH:/usr/local/kubebuilder/bin

export IMAGE_NAME_AND_VERSION=${1}
KUBEBUILDER_ASSETS="$(pwd)/kubebuilder_1.0.8_linux_amd64/bin"
export KUBEBUILDER_ASSETS
make test
