# Copyright 2019, 2020 IBM Corporation.
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


# Image URL to use all building/pushing image targets;
# Use your own docker registry and image name for dev/test by overridding the IMG and REGISTRY environment variable.
IMG ?= $(shell cat COMPONENT_NAME 2> /dev/null)
REGISTRY ?= quay.io/open-cluster-management

# Github host to use for checking the source tree;
# Override this variable ue with your own value if you're working on forked repo.
GIT_HOST ?= github.com/open-cluster-management

PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"

# Keep an existing GOPATH, make a private one if it is undefined
# GOPATH_DEFAULT := $(PWD)/.go
# export GOPATH ?= $(GOPATH_DEFAULT)
TESTARGS_DEFAULT := "-v"
export TESTARGS ?= $(TESTARGS_DEFAULT)
DEST := $(GOPATH)/src/$(GIT_HOST)/$(BASE_DIR)
VERSION ?= $(shell cat COMPONENT_VERSION 2> /dev/null)
BINDIR ?= output
BUILD_LDFLAGS  = $(shell hack/version.sh $(BASE_DIR) $(GIT_HOST)/$(BASE_DIR))
TYPES_FILES = $(shell find pkg/apis -name types.go)

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
    TARGET_OS ?= linux
    XARGS_FLAGS="-r"
else ifeq ($(LOCAL_OS),Darwin)
    TARGET_OS ?= darwin
    XARGS_FLAGS=
else
    $(error "This system's OS $(LOCAL_OS) isn't recognized/supported")
endif

# Use podman if available, otherwise use docker
ifeq ($(CONTAINER_ENGINE),)
	CONTAINER_ENGINE = $(shell podman version > /dev/null && echo podman || echo docker)
endif

.PHONY: fmt lint test coverage build images build-push-images

# GITHUB_USER containing '@' char must be escaped with '%40'
GITHUB_USER := $(shell echo $(GITHUB_USER) | sed 's/@/%40/g')
GITHUB_TOKEN ?=

USE_VENDORIZED_BUILD_HARNESS ?=

ifndef USE_VENDORIZED_BUILD_HARNESS
-include $(shell curl -s -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Accept: application/vnd.github.v4.raw' -L https://api.github.com/repos/open-cluster-management/build-harness-extensions/contents/templates/Makefile.build-harness-bootstrap -o .build-harness-bootstrap; echo .build-harness-bootstrap)
else
-include vbh/.build-harness-bootstrap
endif

default::
	@echo "Build Harness Bootstrapped"

include common/Makefile.common.mk
include test/e2e/Makefile.e2e.mk


############################################################
# copyright check section
############################################################

copyright-check:
	./build/copyright-check.sh $(TRAVIS_BRANCH) $(TRAVIS_PULL_REQUEST_BRANCH)


############################################################
# format section
############################################################

fmt: format-go

############################################################
# lint section
############################################################

lint: lint-all

############################################################
# test section
############################################################

test:
	@go test ${TESTARGS} $(shell go list ./... | grep -v /test/)

############################################################
# e2e test section
############################################################

e2e-test: run-all-e2e-test

############################################################
# coverage section
############################################################

coverage:
	@common/scripts/codecov.sh

############################################################
# This section contains the code generation stuff
############################################################

generate_exes: $(BINDIR)/defaulter-gen \
  $(BINDIR)/deepcopy-gen \
  $(BINDIR)/conversion-gen \
  $(BINDIR)/client-gen \
  $(BINDIR)/lister-gen \
  $(BINDIR)/informer-gen \
  $(BINDIR)/openapi-gen \
  $(BINDIR)/go-to-protobuf \
  $(BINDIR)/protoc-gen-gogo \

$(BINDIR)/defaulter-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/defaulter-gen

$(BINDIR)/deepcopy-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/deepcopy-gen

$(BINDIR)/conversion-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/conversion-gen

$(BINDIR)/client-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/client-gen

$(BINDIR)/lister-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/lister-gen

$(BINDIR)/informer-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/informer-gen

$(BINDIR)/openapi-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/openapi-gen

$(BINDIR)/go-to-protobuf:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/go-to-protobuf

$(BINDIR)/protoc-gen-gogo:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo

# Regenerate all files if the gen exes changed or any "types.go" files changed
generate_files: generate_exes $(TYPES_FILES)
  # generate apiserver deps
	hack/update-apiserver-gen.sh
  # generate protobuf
	hack/update-protobuf.sh
  # generate all pkg/client contents
	hack/update-client-gen.sh

 generate_proxyserver_files: generate_exes $(TYPES_FILES)
	# generate apiserver deps
	hack/proxyserver/update-apiserver-gen.sh
	# generate protobuf
	hack/proxyserver/update-protobuf.sh

############################################################
# build section
############################################################

build: acm-proxyserver mcm-apiserver mcm-webhook mcm-controller klusterlet klusterlet-connectionmanager serviceregistry acm-agent acm-controller


mcm-apiserver:
	@common/scripts/gobuild.sh $(BINDIR)/mcm-apiserver -ldflags '-s -w -X $(SC_PKG)/pkg.VERSION=$(VERSION) $(BUILD_LDFLAGS)' ./cmd/mcm-apiserver

mcm-webhook:
	@common/scripts/gobuild.sh $(BINDIR)/mcm-webhook ./cmd/mcm-webhook/

mcm-controller:
	@common/scripts/gobuild.sh $(BINDIR)/mcm-controller ./cmd/mcm-controller

klusterlet:
	@common/scripts/gobuild.sh $(BINDIR)/klusterlet -ldflags '-s -w  $(BUILD_LDFLAGS)' ./cmd/klusterlet

klusterlet-connectionmanager:
	@common/scripts/gobuild.sh $(BINDIR)/klusterlet-connectionmanager ./cmd/klusterlet-connectionmanager

serviceregistry:
	@common/scripts/gobuild.sh $(BINDIR)/serviceregistry ./cmd/serviceregistry

acm-controller:
	@common/scripts/gobuild.sh $(BINDIR)/acm-controller github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-controller

acm-agent:
	@common/scripts/gobuild.sh $(BINDIR)/acm-agent ./cmd/acm-agent


acm-proxyserver:
	@common/scripts/gobuild.sh $(BINDIR)/acm-proxyserver -ldflags '-s -w -X $(SC_PKG)/pkg.VERSION=$(VERSION) $(BUILD_LDFLAGS)' github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-proxyserver

############################################################
# images section
############################################################

build-images:
	@$(CONTAINER_ENGINE) build . -f Dockerfile -t ${IMAGE_NAME_AND_VERSION}
	@$(CONTAINER_ENGINE) tag ${IMAGE_NAME_AND_VERSION} $(REGISTRY)/$(IMG):latest

############################################################
# clean section
############################################################
clean::
	rm -rf $(BINDIR)

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/action/v1beta1" output:crd:artifacts:config=deploy/dev/hub/resources/crds
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/view/v1beta1" output:crd:artifacts:config=deploy/dev/hub/resources/crds
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/inventory/v1alpha1" output:crd:artifacts:config=deploy/dev/hub/resources/crds

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/action/v1beta1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/view/v1beta1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/inventory/v1alpha1"

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif