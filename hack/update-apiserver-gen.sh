#!/bin/bash

# licensed Materials - Property of IBM
# 5737-E67
# (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
# US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# The only argument this script should ever be called with is '--verify-only'

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

realpath() {
    [[ $1 = /* ]] && echo "$1" || echo "$PWD/${1#./}"
}

REPO_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")"/..)
BINDIR="${REPO_ROOT}"/output
SC_PKG='github.com/open-cluster-management/multicloud-operators-foundation'

# Generate defaults
"${BINDIR}"/defaulter-gen "$@" \
	 --v 1 --logtostderr \
	 --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	 --input-dirs "${SC_PKG}/pkg/apis/mcm" \
   --input-dirs "${SC_PKG}/pkg/apis/mcm/v1alpha1" \
   --input-dirs "${SC_PKG}/pkg/apis/mcm/v1beta1" \
	 --extra-peer-dirs "${SC_PKG}/pkg/apis/mcm" \
	 --extra-peer-dirs "${SC_PKG}/pkg/apis/mcm/v1alpha1" \
   --extra-peer-dirs "${SC_PKG}/pkg/apis/mcm/v1beta1" \
	 --output-file-base "zz_generated.defaults"
# Generate deep copies
"${BINDIR}"/deepcopy-gen "$@" \
	 --v 1 --logtostderr\
	 --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	 --input-dirs "${SC_PKG}/pkg/apis/mcm" \
   --input-dirs "${SC_PKG}/pkg/apis/mcm/v1alpha1" \
   --input-dirs "${SC_PKG}/pkg/apis/mcm/v1beta1" \
	 --output-file-base zz_generated.deepcopy
# Generate conversions
"${BINDIR}"/conversion-gen "$@" \
	 --v 1 --logtostderr \
	 --extra-peer-dirs k8s.io/api/core/v1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
	 --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	 --input-dirs "${SC_PKG}/pkg/apis/mcm" \
   --input-dirs "${SC_PKG}/pkg/apis/mcm/v1beta1" \
   --input-dirs "${SC_PKG}/pkg/apis/mcm/v1alpha1" \
	 --output-file-base zz_generated.conversion

# generate openapi for servicecatalog and settings group
"${BINDIR}"/openapi-gen "$@" \
	--v 1 --logtostderr \
	--go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	--input-dirs "${SC_PKG}/pkg/apis/mcm/v1alpha1,${SC_PKG}/pkg/apis/mcm/v1beta1,k8s.io/api/core/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/version,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/util/intstr,k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1,k8s.io/api/certificates/v1beta1" \
	--output-package "${SC_PKG}/pkg/apis/mcm/openapi" \
  --report-filename ".api_violation.report"
