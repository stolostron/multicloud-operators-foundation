#!/bin/bash

# Run our build target and set IMAGE_NAME_AND_VERSION
export IMAGE_NAME_AND_VERSION=${1}
make build-images
