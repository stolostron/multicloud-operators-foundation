CURL := $(shell which curl)
TMP ?= /tmp
JQ_VERSION ?= 1.5
JQ_PLATFORM ?= $(shell echo $(BUILD_HARNESS_OS) | sed 's/darwin/osx-amd/g')
JQ_ARCH ?= $(shell echo $(BUILD_HARNESS_ARCH) | sed 's/amd64/64/g')
JQ_URL ?= https://github.com/stedolan/jq/releases/download/jq-$(JQ_VERSION)/jq-$(JQ_PLATFORM)$(JQ_ARCH)
JQ ?= $(BUILD_HARNESS_PATH)/vendor/jq

.PHONY: jq/install
## Install jq
jq/install: %install:
	@[ -x $(JQ) ] || ( \
		echo "Installing JQ $(JQ_VERSION) ($(JQ_PLATFORM)$(JQ_ARCH)) from $(JQ_URL)" && \
		curl '-#' -fL -o $(JQ) $(JQ_URL) && \
		chmod +x $(JQ) \
		)
	$(JQ) --version

.PHONY: jq/search
## Pipe a JSON file to jq to process filter defined in environment variable JQ_SEARCH_VAR
jq/search: %jq/search:
	@$(shell $(SELF) jq/install 1&>2 /dev/null)
	@$(JQ) $(JQ_SEARCH_VAR)
