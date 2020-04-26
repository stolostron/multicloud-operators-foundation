#!/bin/bash

# Run unit test
export IMAGE_NAME_AND_VERSION=${1}
KUBEBUILDER_ASSETS="$(pwd)/kubebuilder_1.0.8_linux_amd64/bin"
export KUBEBUILDER_ASSETS
make test
