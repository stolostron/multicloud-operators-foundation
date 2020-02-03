#!/bin/bash

# licensed Materials - Property of IBM
# 5737-E67
# (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
# US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.# Copyright 2015 The Kubernetes Authors.
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

# The contents of this file are in a specific order
# Listers depend on the base client
# Informers depend on listers
set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

realpath() {
    [[ $1 = /* ]] && echo "$1" || echo "$PWD/${1#./}"
}

REPO_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")"/..)
BINDIR=${REPO_ROOT}/output

# Generate the internal clientset (pkg/client/clientset_generated/internalclientset)
"${BINDIR}"/client-gen "$@" \
	      --input-base github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis \
	      --input mcm/ \
	      --clientset-path github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated \
	      --clientset-name internalclientset \
	      --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt
# Generate the versioned clientset (pkg/client/clientset_generated/clientset)
"${BINDIR}"/client-gen "$@" \
        --input-base github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis \
        --input "mcm/v1alpha1,mcm/v1beta1" \
        --clientset-path github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated \
        --clientset-name clientset \
        --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt
# generate listers after having the base client generated, and before informers
"${BINDIR}"/lister-gen "$@" \
	      --input-dirs="github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm,github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1beta1,github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1" \
	      --output-package "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/listers_generated" \
	      --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt
# generate informers after the listers have been generated
"${BINDIR}"/informer-gen "$@" \
	      --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	      --input-dirs "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm,github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1beta1,github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1" \
	      --internal-clientset-package "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/internalclientset" \
	      --versioned-clientset-package "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset" \
	      --listers-package "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/listers_generated" \
	      --output-package "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated"

# Generate the versioned clientset (pkg/client/clientset_generated/clientset)
"${BINDIR}"/client-gen "$@" \
        --input-base k8s.io/cluster-registry/pkg/apis\
        --input clusterregistry/v1alpha1 \
        --clientset-path github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_clientset_generated \
        --clientset-name clientset \
        --go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt
# generate listers after having the base client generated, and before informers
"${BINDIR}"/lister-gen "$@" \
        	--input-dirs="k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1" \
        	--output-package "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_listers_generated" \
        	--go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt
# generate informers after the listers have been generated
"${BINDIR}"/informer-gen "$@" \
        	--go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
        	--input-dirs "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1" \
        	--versioned-clientset-package "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset" \
        	--listers-package "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_listers_generated" \
        	--output-package "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated"
