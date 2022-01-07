all: build
.PHONY: all

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/deps.mk \
	targets/openshift/imagebuilder.mk \
	targets/openshift/images.mk \
	targets/openshift/bindata.mk \
	lib/tmp.mk \
)

# Tools for deploy
KUBECONFIG ?= ./.kubeconfig
KUBECTL?=kubectl
KUSTOMIZE?=$(PERMANENT_TMP_GOPATH)/bin/kustomize
KUSTOMIZE_VERSION?=v3.5.4
KUSTOMIZE_ARCHIVE_NAME?=kustomize_$(KUSTOMIZE_VERSION)_$(GOHOSTOS)_$(GOHOSTARCH).tar.gz
kustomize_dir:=$(dir $(KUSTOMIZE))


# Image URL to use all building/pushing image targets;
IMAGE ?= multicloud-manager
IMAGE_REGISTRY ?= quay.io/stolostron
IMAGE_TAG ?= latest
FOUNDATION_IMAGE_NAME ?= $(IMAGE_REGISTRY)/$(IMAGE):$(IMAGE_TAG)

GIT_HOST ?= github.com/stolostron
BASE_DIR := $(shell basename $(PWD))
DEST := $(GOPATH)/src/$(GIT_HOST)/$(BASE_DIR)
BINDIR ?= _output

SED_CMD:=sed
ifeq ($(GOHOSTOS),darwin)
	ifeq ($(GOHOSTARCH),amd64)
		SED_CMD:=gsed
	endif
endif

# Controller runtime need use this variable as tmp cache dir. if not set, ut will fail in prow
export XDG_CACHE_HOME ?= $(BASE_DIR)/.cache

# KUBEBUILDER for unit test
export KUBEBUILDER_ASSETS ?=$(shell pwd)/$(PERMANENT_TMP_GOPATH)/kubebuilder/bin

K8S_VERSION ?=1.16.4
KB_TOOLS_ARCHIVE_NAME :=kubebuilder-tools-$(K8S_VERSION)-$(GOHOSTOS)-$(GOHOSTARCH).tar.gz
KB_TOOLS_ARCHIVE_PATH := $(PERMANENT_TMP_GOPATH)/$(KB_TOOLS_ARCHIVE_NAME)

# Add packages to do unit test
GO_TEST_PACKAGES :=./pkg/...

CRD_OPTIONS ?= "crd:crdVersions=v1"

# This will call a macro called "build-image" which will generate image specific targets based on the parameters:
# $0 - macro name
# $1 - target suffix
# $2 - Dockerfile path
# $3 - context directory for image build
# It will generate target "image-$(1)" for building the image and binding it as a prerequisite to target "images".
$(call build-image,$(IMAGE),$(IMAGE_REGISTRY)/$(IMAGE),./Dockerfile,.)

test-unit: ensure-kubebuilder

deploy-hub:
	deploy/managedcluster/hub/install.sh

deploy-klusterlet:
	deploy/managedcluster/klusterlet/install.sh

deploy-foundation: ensure-kustomize
	cp deploy/foundation/hub/kustomization.yaml deploy/foundation/hub/kustomization.yaml.tmp
	cd deploy/foundation/hub && ../../../$(KUSTOMIZE) edit set image 'quay.io/stolostron/multicloud-manager'=$(FOUNDATION_IMAGE_NAME)
	$(SED_CMD) -i.tmp "s,quay.io/stolostron/multicloud-manager,$(FOUNDATION_IMAGE_NAME)," deploy/foundation/hub/patches.yaml
	$(KUSTOMIZE) build deploy/foundation/hub | $(KUBECTL) apply -f -
	mv deploy/foundation/hub/kustomization.yaml.tmp deploy/foundation/hub/kustomization.yaml
	mv deploy/foundation/hub/patches.yaml.tmp deploy/foundation/hub/patches.yaml

clean-foundation-agent:
	$(KUBECTL) get managedclusteraddons -A | grep work-manager | awk '{print $$1" "$$2}' | xargs -n 2 $(KUBECTL) delete managedclusteraddons -n

clean-foundation-hub:
	$(KUBECTL) delete -k deploy/foundation/hub

clean-foundation: clean-foundation-hub clean-foundation-agent

build-e2e:
	go test -c ./test/e2e

test-e2e: build-e2e deploy-hub deploy-klusterlet deploy-foundation
	deploy/foundation/scripts/install-check.sh
	./e2e.test -test.v -ginkgo.v

############################################################
# This section contains the code generation stuff
############################################################

generate_exes: $(BINDIR)/defaulter-gen \
  $(BINDIR)/deepcopy-gen \
  $(BINDIR)/conversion-gen \
  $(BINDIR)/openapi-gen \
  $(BINDIR)/go-to-protobuf \
  $(BINDIR)/protoc-gen-gogo \
  
$(BINDIR)/defaulter-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/defaulter-gen

$(BINDIR)/deepcopy-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/deepcopy-gen

$(BINDIR)/conversion-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/conversion-gen

$(BINDIR)/openapi-gen:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/openapi-gen

$(BINDIR)/go-to-protobuf:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/go-to-protobuf

$(BINDIR)/protoc-gen-gogo:
	go build -o $@ $(DEST)/vendor/k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo


# Regenerate all files if the gen exes changed or any "types.go" files changed
generate_files: generate_exes
  # generate apiserver deps
	hack/update-apiserver-gen.sh
  # generate protobuf
	hack/update-protobuf.sh


# Generate manifests e.g. CRD, RBAC etc.
manifests: ensure-controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/action/v1beta1" output:crd:artifacts:config=deploy/foundation/hub/resources/crds
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/view/v1beta1" output:crd:artifacts:config=deploy/foundation/hub/resources/crds
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/internal.open-cluster-management.io/v1beta1" output:crd:artifacts:config=deploy/foundation/hub/resources/crds
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/inventory/v1alpha1" output:crd:artifacts:config=deploy/foundation/hub/resources/crds
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/apis/imageregistry/v1alpha1" output:crd:artifacts:config=deploy/foundation/hub/resources/crds

# Generate code
generate: ensure-controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/action/v1beta1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/view/v1beta1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/inventory/v1alpha1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/imageregistry/v1alpha1"
	$(CONTROLLER_GEN) object:headerFile="hack/custom-boilerplate.go.txt" paths="./pkg/apis/internal.open-cluster-management.io/v1beta1"

# Ensure kubebuilder
ensure-kubebuilder:
ifeq "" "$(wildcard $(KUBEBUILDER_ASSETS))"
	$(info Downloading kube-apiserver into '$(KUBEBUILDER_ASSETS)')
	mkdir -p '$(KUBEBUILDER_ASSETS)'
	curl -s -f -L https://storage.googleapis.com/kubebuilder-tools/$(KB_TOOLS_ARCHIVE_NAME) -o '$(KB_TOOLS_ARCHIVE_PATH)'
	tar -C '$(KUBEBUILDER_ASSETS)' --strip-components=2 -zvxf '$(KB_TOOLS_ARCHIVE_PATH)'
else
	$(info Using existing kube-apiserver from "$(KUBEBUILDER_ASSETS)")
endif

# Ensure kustomize
ensure-kustomize:
ifeq "" "$(wildcard $(KUSTOMIZE))"
	$(info Installing kustomize into '$(KUSTOMIZE)')
	mkdir -p '$(kustomize_dir)'
	curl -s -f -L https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F$(KUSTOMIZE_VERSION)/$(KUSTOMIZE_ARCHIVE_NAME) -o '$(kustomize_dir)$(KUSTOMIZE_ARCHIVE_NAME)'
	tar -C '$(kustomize_dir)' -zvxf '$(kustomize_dir)$(KUSTOMIZE_ARCHIVE_NAME)'
	chmod +x '$(KUSTOMIZE)';
else
	$(info Using existing kustomize from "$(KUSTOMIZE)")
endif

# Ensure controller-gen
ensure-controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
