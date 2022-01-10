#!/bin/bash

# Copyright (c) 2020 Red Hat, Inc.


# The only argument this script should ever be called with is '--verify-only'

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

realpath() {
    [[ $1 = /* ]] && echo "$1" || echo "$PWD/${1#./}"
}

REPO_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")"/..)
BINDIR="${REPO_ROOT}"/_output
SC_PKG='github.com/stolostron/multicloud-operators-foundation'

if [[ "$(protoc --version)" != "libprotoc 3.0."* ]]; then
  echo "Generating protobuf requires protoc 3.0.x. Please download and
install the platform appropriate Protobuf package for your OS:
  https://github.com/google/protobuf/releases"
  exit 1
fi

PATH="$PATH:$BINDIR" go-to-protobuf \
  --output-base="${GOPATH}/src" \
  --apimachinery-packages='-k8s.io/apimachinery/pkg/util/intstr,-k8s.io/apimachinery/pkg/api/resource,-k8s.io/apimachinery/pkg/runtime/schema,-k8s.io/apimachinery/pkg/runtime,-k8s.io/apimachinery/pkg/apis/meta/v1,-k8s.io/apimachinery/pkg/apis/meta/v1beta1,-k8s.io/api/core/v1,-k8s.io/api/rbac/v1,-k8s.io/api/certificates/v1beta1' \
  --go-header-file="${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
  --proto-import="${REPO_ROOT}"/vendor \
  --proto-import="${REPO_ROOT}"/third_party/protobuf \
  --packages="${SC_PKG}/pkg/proxyserver/apis/proxy/v1beta1"
