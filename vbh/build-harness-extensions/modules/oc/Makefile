# A simple build harness module to install the newest version of the oc cli.

OC_BUILD_VERSION?=latest
OC_SOURCE_URL?="https://mirror.openshift.com/pub/openshift-v4/clients/oc/${OC_BUILD_VERSION}/linux/oc.tar.gz"

OC_DEST_PATH?=${BUILD_HARNESS_PATH}/vendor
OC_TAR_PATH?=${OC_DEST_PATH}/oc.tar.gz
OC?=${OC_DEST_PATH}/oc
OC_CLUSTER_USER ?=
OC_CLUSTER_PASS ?=
OC_CLUSTER_URL ?=
OC_LOGIN_OPTIONS ?= --insecure-skip-tls-verify=true
OC_COMMAND ?=
OC_LOGGED_IN ?= $(OC_DEST_PATH)/.oc_cli_logged_in

.PHONY: oc/install
## Install the oc cli
oc/install: %install:
	@[ -x $(OC) ] || ( wget -O ${OC_TAR_PATH} ${OC_SOURCE_URL} && \
	tar -xf ${OC_TAR_PATH} -C ${OC_DEST_PATH} && \
	rm -f ${OC_TAR_PATH} )

.PHONY: oc/login
## Login to the OC Environment
oc/login: %login: %install
	@[ -e $(OC_LOGGED_IN) ] || ( $(OC) login $(OC_CLUSTER_URL) $(OC_LOGIN_OPTIONS) -u $(OC_CLUSTER_USER) -p $(OC_CLUSTER_PASS) && touch $(OC_LOGGED_IN) )

.PHONY: oc/command
## Run $(OC_COMMAND) from CLI
oc/command: %command: %install %login
	$(OC) $(OC_COMMAND) 
	@if [ "$(OC_COMMAND)" = "logout" ]; \
	then rm -f $(OC_LOGGED_IN); \
	fi
