#!/bin/bash

set -o errexi
set -o nounset
set -o pipefail
set -o xtrace

# Run e2e test
export IMAGE_NAME_AND_VERSION=${1}


GO111MODULE="on" go get sigs.k8s.io/kind@v0.7.0

kind create cluster
