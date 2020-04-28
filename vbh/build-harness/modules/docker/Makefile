# Docker client
DOCKER := $(shell which docker)

# Jot is for OSX, and shuf is for Linux
RANDOM_PORT ?= $(shell jot -r 1  2000 65000 2>/dev/null || shuf -i 2000-65000 -n 1 2>/dev/null)

DOCKER_REGISTRY ?= index.docker.io
DOCKER_NAMESPACE ?=

DOCKER_CLIENT_VERSION = $(shell $(DOCKER) version --format={{.Client.Version}} 2>/dev/null)
DOCKER_SERVER_VERSION = $(shell $(DOCKER) version --format={{.Server.Version}} 2>/dev/null)

# These vars are are used by the `login` target.
DOCKER_USER ?=
DOCKER_PASS ?=

# The image we're building - defaults to the current directory name
DOCKER_IMAGE ?= $(subst -docker,,$(shell basename "`pwd`"))
DOCKER_IMAGE_ARCH = $(DOCKER_IMAGE)-$(BUILD_HARNESS_ARCH)

# Tag used when building image
DOCKER_BUILD_TAG ?= $(SEMVERSION)

# Tag used when tagging image built with DOCKER_BUILD_TAG and tag pushed to repo
DOCKER_TAG ?= $(DOCKER_BUILD_TAG)

# Complete URI to docker image
DOCKER_URI ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(DOCKER_IMAGE):$(DOCKER_TAG)
DOCKER_ARCH_URI ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(DOCKER_IMAGE_ARCH):$(DOCKER_TAG)

# Filename used for docker export
DOCKER_EXPORT ?= $(DOCKER_IMAGE)-$(DOCKER_TAG)-export.tar

# Path to build (where the Dockerfile is located)
DOCKER_BUILD_PATH ?= .

# The default dockerfile name used
ifeq ($(findstring ppc64le,$(BUILD_HARNESS_ARCH)),ppc64le)
DOCKER_FILE_SUFFIX := .ppc64le
endif
ifeq ($(findstring s390x,$(BUILD_HARNESS_ARCH)),s390x)
DOCKER_FILE_SUFFIX := .s390x
endif
DOCKER_FILE ?= Dockerfile$(DOCKER_FILE_SUFFIX)

# If attempting to start the container, this name will be used
DOCKER_CONTAINER_NAME ?= test_$(DOCKER_IMAGE)

# When specifying `DOCKER_NETWORK` as desired network name, `DOCKER_NETWORK_OP` should also be specified as `--network`
# When specifying `DOCKER_IP` as desired IP address, `DOCKER_IP_OP` should also be specified as `--ip`
DOCKER_BIND_PORT ?= $(RANDOM_PORT):80
DOCKER_NETWORK_OP ?=
DOCKER_NETWORK ?=
DOCKER_IP_OP ?=
DOCKER_IP ?=
DOCKER_SHELL ?= /bin/bash

# Arguments passed to "docker build"
DOCKER_BUILD_OPTS ?=

# Arguments passed to "docker run"
DOCKER_RUN_OPTS ?=

# If specifying a custom DOCKER_BUILD_UTILITY, ensure it is quoted.
ifndef BUILD_HARNESS_BRANCH
DOCKER_BUILD_UTILITY ?= "build-harness: dev-local"
else
DOCKER_BUILD_UTILITY ?= "build-harness: $(BUILD_HARNESS_BRANCH)"
endif

MANIFEST_TOOL ?= $(BUILD_HARNESS_PATH)/vendor/manifest-tool
MANIFEST_TOOL_VERSION ?= v0.7.0
MANIFEST_TOOL_OS ?= $(BUILD_HARNESS_OS)
MANIFEST_TOOL_ARCH ?= $(BUILD_HARNESS_ARCH)
MANIFEST_TOOL_URL ?= https://github.com/estesp/manifest-tool/releases/download/$(MANIFEST_TOOL_VERSION)/manifest-tool-$(MANIFEST_TOOL_OS)-$(MANIFEST_TOOL_ARCH)

DOCKER_MANIFEST_TEMPLATE ?= $(BUILD_HARNESS_PATH)/modules/docker/manifest.yaml

DOCKER_FROM_USER ?= $(DOCKER_USER)
DOCKER_FROM_PASS ?= $(DOCKER_PASS)
DOCKER_FROM_REGISTRY ?= $(DOCKER_REGISTRY)
DOCKER_FROM_NAMESPACE ?= $(DOCKER_NAMESPACE)
DOCKER_FROM_IMAGE ?= $(DOCKER_IMAGE)
DOCKER_FROM_TAG ?= $(DOCKER_TAG)
DOCKER_TO_USER ?= $(DOCKER_USER)
DOCKER_TO_PASS ?= $(DOCKER_PASS)
DOCKER_TO_REGISTRY ?= $(DOCKER_FROM_REGISTRY)
DOCKER_TO_NAMESPACE ?= $(DOCKER_FROM_NAMESPACE)
DOCKER_TO_IMAGE ?= $(DOCKER_FROM_IMAGE)
DOCKER_TO_TAG ?= $(DOCKER_FROM_TAG)
DOCKER_ADDTL_TAG ?=
DOCKER_IMAGE_AND_TAG ?= $(DOCKER_IMAGE):$(DOCKER_BUILD_TAG)


.PHONY : docker/deps docker/info docker/build docker/push docker/pull docker/clean docker/run docker/shell docker/attach docker/update docker/start docker/stop docker/rm docker/logs docker/manifest-tool docker/enable-experimental-cli docker/multi-arch docker/copy

docker/deps:
	$(call assert_set DOCKER)
	@[ -x $(DOCKER) ] || (echo "$(DOCKER) not executable"; exit 1)

## Display info about the docker environment
docker/info:
	@$(SELF) docker/deps
	@echo "DOCKER=$(DOCKER)"
	@echo "DOCKER_IMAGE=$(DOCKER_IMAGE)"
	@echo "DOCKER_IMAGE_ARCH=$(DOCKER_IMAGE_ARCH)"
	@echo "DOCKER_BUILD_TAG=$(DOCKER_BUILD_TAG)"
	@echo "DOCKER_TAG=$(DOCKER_TAG)"
	@echo "DOCKER_CONTAINER_NAME=$(DOCKER_CONTAINER_NAME)"
	@echo "DOCKER_BUILD_OPTS=$(DOCKER_BUILD_OPTS)"
	@echo "DOCKER_RUN_OPTS=$(DOCKER_RUN_OPTS)"
	@echo "DOCKER_NETWORK=$(DOCKER_NETWORK_OP) $(DOCKER_NETWORK)"
	@echo "DOCKER_IP=$(DOCKER_IP_OP) $(DOCKER_IP)"
	@echo "DOCKER_FILE=$(DOCKER_FILE)"
	@echo "DOCKER_REGISTRY=$(DOCKER_REGISTRY)"
	@echo "DOCKER_NAMESPACE=$(DOCKER_NAMESPACE)"
	@echo "DOCKER_URI=$(DOCKER_URI)"
	@echo "DOCKER_ARCH_URI=$(DOCKER_ARCH_URI)"
	@echo "DOCKER_CLIENT_VERSION=$(DOCKER_CLIENT_VERSION)"
	@echo "DOCKER_SERVER_VERSION=$(DOCKER_SERVER_VERSION)"

## Build a docker image
docker/build:
	@$(SELF) docker/deps
	@echo "INFO: Building $(DOCKER_IMAGE_AND_TAG) using $(DOCKER_BUILD_PATH)/$(DOCKER_FILE) on docker $(DOCKER_SERVER_VERSION) $(DOCKER_BUILD_OPTS)"
	@cd $(DOCKER_BUILD_PATH) && cp $(DOCKER_FILE) $(DOCKER_FILE).tmp && echo -e '\nLABEL build-utility=$(DOCKER_BUILD_UTILITY)' >> $(DOCKER_FILE).tmp && $(DOCKER) build $(DOCKER_BUILD_OPTS) -t "$(DOCKER_IMAGE_AND_TAG)" -f $(DOCKER_FILE).tmp . && rm $(DOCKER_FILE).tmp

## Push image to Docker Hub
docker/push:
	@$(SELF) docker/deps
	@echo "INFO: Pushing $(DOCKER_URI)"
	@until $(DOCKER) push "$(DOCKER_URI)"; do sleep 1; done

## Push the architecture-specific image to a Docker registry
docker/push-arch:
	@$(SELF) docker/deps
	@echo "INFO: Pushing $(DOCKER_ARCH_URI)"
	@until $(DOCKER) push "$(DOCKER_ARCH_URI)"; do sleep 1; done

## Pull docker image from Docker Hub
docker/pull:
	@$(SELF) docker/deps
	@echo "INFO: Pulling $(DOCKER_URI)"
	@$(DOCKER) pull "$(DOCKER_URI)"

## Test docker image
docker/test:
	@$(DOCKER) version 2>&1 >/dev/null | grep 'Error response from daemon'; [ $$? -ne 0 ]
	@echo "OK"

## Tag the last built image with an architecture-specific`DOCKER_TAG`
docker/tag:
	@$(SELF) docker/deps
	@echo INFO: Tagging $(DOCKER_IMAGE_AND_TAG) as $(DOCKER_URI)
ifeq ($(findstring 1.9.1,$(DOCKER_SERVER_VERSION)),1.9.1)
	@$(DOCKER) tag -f "$(DOCKER_IMAGE_AND_TAG)" "$(DOCKER_URI)"
else
	@$(DOCKER) tag "$(DOCKER_IMAGE_AND_TAG)" "$(DOCKER_URI)"
endif

## Tag the last built image with `DOCKER_TAG`
docker/tag-arch:
	@$(SELF) docker/deps
	@echo INFO: Tagging $(DOCKER_IMAGE_AND_TAG) as $(DOCKER_ARCH_URI)
ifeq ($(findstring 1.9.1,$(DOCKER_SERVER_VERSION)),1.9.1)
	@$(DOCKER) tag -f "$(DOCKER_IMAGE_AND_TAG)" "$(DOCKER_ARCH_URI)"
else
	@$(DOCKER) tag "$(DOCKER_IMAGE_AND_TAG)" "$(DOCKER_ARCH_URI)"
endif

## Remove existing docker images
docker/clean:
	@$(SELF) docker/deps
	@echo INFO: Clean $(DOCKER_IMAGE_AND_TAG), $(DOCKER_URI)
	@$(DOCKER) rmi -f "$(DOCKER_IMAGE_AND_TAG)"
	@$(DOCKER) rmi -f "$(DOCKER_URI)"
	$(eval DOCKER_BUILD_OPTS += --no-cache=true)

## Test drive the image
docker/run:
	@$(SELF) docker/deps
	@echo "INFO: Running $(DOCKER_IMAGE_AND_TAG) as $(DOCKER_CONTAINER_NAME) with $(DOCKER_RUN_OPTS)" 
	@$(DOCKER) run $(DOCKER_RUN_OPTS) --name "$(DOCKER_CONTAINER_NAME)" $(DOCKER_NETWORK_OP) $(DOCKER_NETWORK) $(DOCKER_IP_OP) $(DOCKER_IP) --rm -p "$(DOCKER_BIND_PORT)" -t -i "$(DOCKER_IMAGE_AND_TAG)"

## Run the container and start a shell
docker/shell:
	@$(SELF) docker/deps
	@echo INFO: Starting shell in $(DOCKER_IMAGE) as $(DOCKER_CONTAINER_NAME) with $(DOCKER_BIND_PORT)
	@$(DOCKER) run --name "$(DOCKER_CONTAINER_NAME)" --rm -p "$(DOCKER_BIND_PORT)" -t -i --volume "$(shell pwd):/opt" --entrypoint="$(DOCKER_SHELL)"  "$(DOCKER_IMAGE_AND_TAG)" -c $(DOCKER_SHELL)

## Attach to the running container
docker/attach:
	@$(SELF) docker/deps
	@echo INFO: Attaching to $(DOCKER_CONTAINER_NAME)
	@$(DOCKER) exec -i -t  "$(DOCKER_CONTAINER_NAME)" $(DOCKER_SHELL)

## Login to docker registry
docker/login:
	@$(SELF) docker/deps
	@$(call assert_set,DOCKER_USER)
	@$(call assert_set,DOCKER_PASS)
	@echo "INFO: Logging in as $(DOCKER_USER)"
	@$(DOCKER) login $(DOCKER_REGISTRY) -u $(DOCKER_USER) -p $(DOCKER_PASS) || $(DOCKER) login -u $(DOCKER_USER) -p $(DOCKER_PASS)

## Export docker images to file
docker/export:
	@$(SELF) docker/deps
	@$(call assert_set,DOCKER_IMAGE)
	@$(call assert_set,DOCKER_TAG)
	@$(call assert_set,DOCKER_EXPORT)
	@echo INFO: Exporting $(DOCKER_IMAGE):$(DOCKER_TAG) to $(DOCKER_EXPORT)
	@$(DOCKER) save $(DOCKER_IMAGE):$(DOCKER_TAG) > $(DOCKER_EXPORT)

## Import docker images from file
docker/import:
	@$(SELF) docker/deps
	@$(call assert_set,DOCKER_EXPORT)
	@echo INFO: Importing $(DOCKER_EXPORT)
	@$(DOCKER) load -i $(DOCKER_EXPORT)

## Download and install the manifest tool so you don't need the edge docker client
docker/manifest-tool:
	@$(SELF) docker/deps
	@[ -x $(MANIFEST_TOOL) ] || ( \
		echo "Installing manifest-tool $(MANIFEST_TOOL_VERSION) ($(MANIFEST_TOOL_OS)-$(MANIFEST_TOOL_ARCH)) from $(MANIFEST_TOOL_URL)" && \
		curl '-#' -fL -o $(MANIFEST_TOOL) $(MANIFEST_TOOL_URL) && \
		chmod +x $(MANIFEST_TOOL) \
		)
	$(MANIFEST_TOOL) --version

## Push the manifest to a Docker registry
docker/multi-arch:
	@$(SELF) docker/deps
	cp $(DOCKER_MANIFEST_TEMPLATE) /tmp/manifest.yaml
	sed -i.bak -e "s|__DOCKER_TAG__|$(DOCKER_TAG)|g" -e "s|__DOCKER_IMAGE__|$(DOCKER_IMAGE)|g" -e "s|__DOCKER_NAMESPACE__|$(DOCKER_NAMESPACE)|g" -e "s|__DOCKER_REGISTRY__|$(DOCKER_REGISTRY)|g" /tmp/manifest.yaml
	$(MANIFEST_TOOL) --debug --username $(DOCKER_USER) --password $(DOCKER_PASS) push from-spec --ignore-missing /tmp/manifest.yaml

## Copy docker image from one location to another (including multi-arch manifest list images)
docker/copy:
	@$(SELF) docker/manifest-tool
	@$(SELF) jq/install
	@$(call assert_set,DOCKER_FROM_USER)
	@$(call assert_set,DOCKER_FROM_PASS)
	@$(call assert_set,DOCKER_FROM_REGISTRY)
	@$(call assert_set,DOCKER_FROM_NAMESPACE)
	@$(call assert_set,DOCKER_FROM_IMAGE)
	@$(call assert_set,DOCKER_FROM_TAG)
	@$(call assert_set,DOCKER_TO_USER)
	@$(call assert_set,DOCKER_TO_PASS)
	@$(call assert_set,DOCKER_TO_REGISTRY)
	@$(call assert_set,DOCKER_TO_NAMESPACE)
	@$(call assert_set,DOCKER_TO_IMAGE)
	@$(call assert_set,DOCKER_TO_TAG)
	$(BUILD_HARNESS_PATH)/modules/docker/bin/copy.sh "$(DOCKER_FROM_USER)" "$(DOCKER_FROM_PASS)" "$(DOCKER_FROM_REGISTRY)" "$(DOCKER_FROM_NAMESPACE)" "$(DOCKER_FROM_IMAGE)" "$(DOCKER_FROM_TAG)" "$(DOCKER_TO_USER)" "$(DOCKER_TO_PASS)" "$(DOCKER_TO_REGISTRY)" "$(DOCKER_TO_NAMESPACE)" "$(DOCKER_TO_IMAGE)" "$(DOCKER_TO_TAG)" "$(DOCKER_ADDTL_TAG)"
