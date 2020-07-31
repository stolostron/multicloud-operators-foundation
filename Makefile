all: build
.PHONY: all

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/deps.mk \
	targets/openshift/images.mk \
	targets/openshift/bindata.mk \
	lib/tmp.mk \
)

# Image URL to use all building/pushing image targets;
# Use your own docker registry and image name for dev/test by overridding the IMG and REGISTRY environment variable.
IMG ?= multicloud-manager
REGISTRY ?= quay.io/open-cluster-management

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"

# Keep an existing GOPATH, make a private one if it is undefined
# GOPATH_DEFAULT := $(PWD)/.go
# export GOPATH ?= $(GOPATH_DEFAULT)
TESTARGS_DEFAULT := "-v"
export TESTARGS ?= $(TESTARGS_DEFAULT)

TYPES_FILES = $(shell find pkg/apis -name types.go)



# This will call a macro called "build-image" which will generate image specific targets based on the parameters:
# $0 - macro name
# $1 - target suffix
# $2 - Dockerfile path
# $3 - context directory for image build
# It will generate target "image-$(1)" for building the image and binding it as a prerequisite to target "images".
$(call build-image,$(IMG),$(REGISTRY)/$(IMG),./Dockerfile,.)


.PHONY: fmt lint test coverage build images build-push-images

# GITHUB_USER containing '@' char must be escaped with '%40'
GITHUB_USER := $(shell echo $(GITHUB_USER) | sed 's/@/%40/g')
GITHUB_TOKEN ?=

include test/e2e/Makefile.e2e.mk

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

############################################################
# This section contains the code generation stuff
############################################################

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/action/v1beta1" output:crd:artifacts:config=deploy/dev/hub/resources/crds
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/view/v1beta1" output:crd:artifacts:config=deploy/dev/hub/resources/crds
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/cluster/v1beta1" output:crd:artifacts:config=deploy/dev/hub/resources/crds
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/inventory/v1alpha1" output:crd:artifacts:config=deploy/dev/hub/resources/crds

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/action/v1beta1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/view/v1beta1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/inventory/v1alpha1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/cluster/v1beta1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/conditions"

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
