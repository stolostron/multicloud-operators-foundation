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

HELM?=$(PERMANENT_TMP_GOPATH)/bin/helm
HELM_VERSION?=v3.14.0
HELM_ARCHIVE_NAME?=helm-$(HELM_VERSION)-$(GOHOSTOS)-$(GOHOSTARCH).tar.gz
helm_dir:=$(dir $(HELM))

CODE_GENENRATOR_VERSION ?= $(shell go list -m k8s.io/code-generator | awk '{print $$2}')

# Image URL to use all building/pushing image targets;
IMAGE ?= multicloud-manager
IMAGE_REGISTRY ?= quay.io/stolostron
IMAGE_TAG ?= latest
FOUNDATION_IMAGE_NAME ?= $(IMAGE_REGISTRY)/$(IMAGE):$(IMAGE_TAG)

GIT_HOST ?= github.com/stolostron
BASE_DIR := $(shell basename $(PWD))

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

K8S_VERSION ?=1.29.1
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

deploy-addons: ensure-helm
	deploy/managedcluster/addons/install.sh

deploy-ocm-controller: ensure-kustomize
	cp deploy/foundation/hub/overlays/ocm-controller/kustomization.yaml deploy/foundation/hub/overlays/ocm-controller/kustomization.yaml.tmp
	cd deploy/foundation/hub/overlays/ocm-controller && ../../../../../$(KUSTOMIZE) edit set image 'quay.io/stolostron/multicloud-manager'=$(FOUNDATION_IMAGE_NAME)
	cp deploy/foundation/hub/overlays/ocm-controller/patch.yaml deploy/foundation/hub/overlays/ocm-controller/patch.yaml.tmp
	$(SED_CMD) -i.tmp "s,quay.io/stolostron/multicloud-manager,$(FOUNDATION_IMAGE_NAME)," deploy/foundation/hub/overlays/ocm-controller/patch.yaml
	$(KUSTOMIZE) build deploy/foundation/hub/overlays/ocm-controller | $(KUBECTL) apply -f -
	mv deploy/foundation/hub/overlays/ocm-controller/kustomization.yaml.tmp deploy/foundation/hub/overlays/ocm-controller/kustomization.yaml
	mv deploy/foundation/hub/overlays/ocm-controller/patch.yaml.tmp deploy/foundation/hub/overlays/ocm-controller/patch.yaml

deploy-foundation: ensure-kustomize
	cp deploy/foundation/hub/overlays/foundation/kustomization.yaml deploy/foundation/hub/overlays/foundation/kustomization.yaml.tmp
	cd deploy/foundation/hub/overlays/foundation && ../../../../../$(KUSTOMIZE) edit set image 'quay.io/stolostron/multicloud-manager'=$(FOUNDATION_IMAGE_NAME)
	cp deploy/foundation/hub/overlays/foundation/patch.yaml deploy/foundation/hub/overlays/foundation/patch.yaml.tmp
	$(SED_CMD) -i.tmp "s,quay.io/stolostron/multicloud-manager,$(FOUNDATION_IMAGE_NAME)," deploy/foundation/hub/overlays/foundation/patch.yaml
	$(KUSTOMIZE) build deploy/foundation/hub/overlays/foundation | $(KUBECTL) apply -f -
	mv deploy/foundation/hub/overlays/foundation/kustomization.yaml.tmp deploy/foundation/hub/overlays/foundation/kustomization.yaml
	mv deploy/foundation/hub/overlays/foundation/patch.yaml.tmp deploy/foundation/hub/overlays/foundation/patch.yaml

clean-foundation-agent:
	$(KUBECTL) get managedclusteraddons -A | grep work-manager | awk '{print $$1" "$$2}' | xargs -n 2 $(KUBECTL) delete managedclusteraddons -n

clean-foundation-hub:
	$(KUBECTL) delete -k deploy/foundation/hub/ocm-controller
	$(KUBECTL) delete -k deploy/foundation/hub/ocm-proxyserver
	$(KUBECTL) delete -k deploy/foundation/hub/ocm-webhook
	$(KUBECTL) delete -k deploy/foundation/hub/rbac
	$(KUBECTL) delete -k deploy/foundation/hub/crds

clean-foundation: clean-foundation-hub clean-foundation-agent

build-e2e:
	go test -c ./test/e2e

test-e2e: build-e2e deploy-hub deploy-klusterlet deploy-foundation deploy-addons
	deploy/foundation/scripts/install-check.sh
	./e2e.test -test.v -ginkgo.v

############################################################
# This section contains the code generation stuff
############################################################

update-scripts:
	hack/update-codegen.sh

update-crds:
	hack/update-crds.sh

update: update-crds update-scripts

verify-crds:
	hack/verify-crds.sh

verify-scripts:
	hack/verify-codegen.sh

verify: verify-crds verify-scripts

update-protobuf:
	go install k8s.io/code-generator/cmd/go-to-protobuf@$(CODE_GENENRATOR_VERSION)
	go install k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo@$(CODE_GENENRATOR_VERSION)
	hack/update-protobuf.sh

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

ensure-helm:
ifeq "" "$(wildcard $(HELM))"
	$(info Installing helm into '$(HELM)')
	mkdir -p '$(helm_dir)'
	curl -s -f -L https://get.helm.sh/$(HELM_ARCHIVE_NAME) -o '$(helm_dir)$(HELM_ARCHIVE_NAME)'
	tar -C '$(helm_dir)' -zvxf '$(helm_dir)$(HELM_ARCHIVE_NAME)'
	mv $(helm_dir)/$(GOHOSTOS)-$(GOHOSTARCH)/helm $(HELM)
	rm -rf $(helm_dir)$(GOHOSTOS)-$(GOHOSTARCH)
	chmod +x '$(HELM)';
else
	$(info Using existing kustomize from "$(HELM)")
endif

include ./test/integration-test.mk
