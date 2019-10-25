# Copyright 2019 The Kubernetes Authors.
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

# This repo is build locally for dev/test by default;
# Override this variable in CI env.
BUILD_LOCALLY ?= 1

# Image URL to use all building/pushing image targets;
# Use your own docker registry and image name for dev/test by overridding the IMG and REGISTRY environment variable.
IMG ?= multicloud-manager
REGISTRY ?= quay.io/multicloudlab

# Github host to use for checking the source tree;
# Override this variable ue with your own value if you're working on forked repo.
GIT_HOST ?= github.ibm.com/IBMPrivateCloud

PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))

# Keep an existing GOPATH, make a private one if it is undefined
GOPATH_DEFAULT := $(PWD)/.go
export GOPATH ?= $(GOPATH_DEFAULT)
GOBIN_DEFAULT := $(GOPATH)/bin
export GOBIN ?= $(GOBIN_DEFAULT)
TESTARGS_DEFAULT := "-v"
export TESTARGS ?= $(TESTARGS_DEFAULT)
DEST := $(GOPATH)/src/$(GIT_HOST)/$(BASE_DIR)
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
                 git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)
BINDIR ?= output
BUILD_LDFLAGS  = $(shell hack/version.sh $(BASE_DIR) $(DEST))
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

.PHONY: all work fmt check coverage lint test build images build-push-images

all: fmt check test coverage build images

ifneq ("$(realpath $(DEST))", "$(realpath $(PWD))")
    $(error Please run 'make' from $(DEST). Current directory is $(PWD))
endif

include common/Makefile.common.mk


############################################################
# work section
############################################################
$(GOBIN):
	@echo "create gobin"
	@mkdir -p $(GOBIN)

work: $(GOBIN)

############################################################
# format section
############################################################

# All available format: format-go format-protos format-python
# Default value will run all formats, override these make target with your requirements:
#    eg: fmt: format-go format-protos
fmt: format-go format-protos format-python

############################################################
# check section
############################################################

check: lint

# All available linters: lint-dockerfiles lint-scripts lint-yaml lint-copyright-banner lint-go lint-python lint-helm lint-markdown lint-sass lint-typescript lint-protos
# Default value will run all linters, override these make target with your requirements:
#    eg: lint: lint-go lint-yaml
lint: lint-all

############################################################
# test section
############################################################

test:
	@go test ${TESTARGS} ./...

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

# Regenerate all files if the gen exes changed or any "types.go" files changed
generate_files: generate_exes $(TYPES_FILES)
  # generate apiserver deps
	hack/update-apiserver-gen.sh
  # generate all pkg/client contents
	hack/update-client-gen.sh


############################################################
# build section
############################################################

build: mcm-apiserver mcm-webhook mcm-controller klusterlet klusterlet-connectionmanager serviceregistry

mcm-apiserver:
	@common/scripts/gobuild.sh $(BINDIR)/mcm-apiserver -ldflags '-s -w -X $(SC_PKG)/pkg.VERSION=$(VERSION) $(BUILD_LDFLAGS)' github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/mcm-apiserver

mcm-webhook:
	@common/scripts/gobuild.sh $(BINDIR)/mcm-webhook github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/mcm-webhook/

mcm-controller:
	@common/scripts/gobuild.sh $(BINDIR)/mcm-controller github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/mcm-controller

klusterlet:
	@common/scripts/gobuild.sh $(BINDIR)/klusterlet -ldflags '-s -w  $(BUILD_LDFLAGS)' github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/klusterlet

klusterlet-connectionmanager:
	@common/scripts/gobuild.sh $(BINDIR)/klusterlet-connectionmanager github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/klusterlet-connectionmanager

serviceregistry:
	@common/scripts/gobuild.sh $(BINDIR)/serviceregistry github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/serviceregistry


############################################################
# images section
############################################################

images: build build-push-images

ifeq ($(BUILD_LOCALLY),0)
    export CONFIG_DOCKER_TARGET = config-docker
endif

build-push-images: $(CONFIG_DOCKER_TARGET)
	@docker build . -f Dockerfile -t $(REGISTRY)/$(IMG):$(VERSION)
	@docker tag $(REGISTRY)/$(IMG):$(VERSION) $(REGISTRY)/$(IMG):latest
	@docker push $(REGISTRY)/$(IMG):$(VERSION)
	@docker push $(REGISTRY)/$(IMG):latest

############################################################
# clean section
############################################################
clean::
	rm -rf $(BINDIR)
