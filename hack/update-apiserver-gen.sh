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

# Generate defaults
"${BINDIR}"/defaulter-gen "$@" \
	 --v 1 --logtostderr \
	 --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	 --input-dirs "${SC_PKG}/pkg/proxyserver/apis/v1beta1" \
	 --extra-peer-dirs "${SC_PKG}/pkg/proxyserver/apis/v1beta1" \
	 --output-file-base "zz_generated.defaults"
# Generate deep copies
"${BINDIR}"/deepcopy-gen "$@" \
	 --v 1 --logtostderr\
	 --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	 --input-dirs "${SC_PKG}/pkg/proxyserver/apis/v1beta1" \
	 --output-file-base zz_generated.deepcopy
# Generate conversions
"${BINDIR}"/conversion-gen "$@" \
	 --v 1 --logtostderr \
	 --extra-peer-dirs k8s.io/api/core/v1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
	 --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	 --input-dirs "${SC_PKG}/pkg/proxyserver/apis/v1beta1" \
	 --output-file-base zz_generated.conversion

# generate openapi for servicecatalog and settings group
"${BINDIR}"/openapi-gen "$@" \
	--v 1 --logtostderr \
	--go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	--input-dirs "${SC_PKG}/pkg/proxyserver/apis/v1beta1,k8s.io/apimachinery/pkg/apis/meta/v1" \
	--output-package "${SC_PKG}/pkg/proxyserver/apis/openapi" \
  --report-filename ".api_violation.report"
