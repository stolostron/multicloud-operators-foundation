CURL := $(shell which curl)
DOCKER := $(shell which docker)

QUAY_TOKEN ?= 
RETAG_DRY_RUN ?= false
RETAG_QUAY_COMPONENT_TAG ?=
RETAG_SNAPSHOT_NAME ?=
RETAG_GITHUB_SHA ?=
RETAG_REPO ?=
PIPELINE_MANIFEST_ORG ?= open-cluster-management

# Retag needs to do the following steps:
#  1. Find an image in quay based on what we would call the team's docker tag, and pull out and assign 'RETAG_QUAY_SHA' from that
#  2. If RETAG_DRY_RUN is set to `true`, then report on the existence or nonexistence of the image; exit with 1 if it doesn't exist 
#  3. If RETAG_DRY_RUN is set to anything other than `true`, derive RETAG_QUAY_SHA using RETAG_QUAY_COMPONENT_TAG from the manifest_digest in quay and tag that image with the snapshot value 'RETAG_SNAPSHOT_NAME'
.PHONY: retag/quay
## Add a tag to an image in quay.io
retag/quay: %quay:
	$(call assert-set,QUAY_TOKEN)
	$(call assert-set,RETAG_SNAPSHOT_NAME)
	$(call assert-set,COMPONENT_NAME)
	$(call assert-set,RETAG_QUAY_COMPONENT_TAG)
	$(call assert-set,PIPELINE_MANIFEST_ORG)
	@$(SELF) jq/install > /dev/null
	@$(eval RETAG_QUAY_SHA := $(shell $(CURL) -X GET -H "Authorization: Bearer ${QUAY_TOKEN}" "https://quay.io/api/v1/repository/${PIPELINE_MANIFEST_ORG}/${COMPONENT_NAME}/tag/?onlyActiveTags=true&specificTag=${RETAG_QUAY_COMPONENT_TAG}" | $(JQ) -r .tags[0].manifest_digest))
	@if [ "${RETAG_QUAY_SHA}" == "null" ] ;then (echo "Tag ${RETAG_QUAY_COMPONENT_TAG} doesn't exist"; exit 1) ; fi
	@if [ "${RETAG_DRY_RUN}" == "true" ]; \
	then echo "Tag ${RETAG_QUAY_COMPONENT_TAG} exists"; \
	else $(CURL) -X PUT -H "Authorization: Bearer ${QUAY_TOKEN}" "https://quay.io/api/v1/repository/${PIPELINE_MANIFEST_ORG}/${COMPONENT_NAME}/tag/${RETAG_SNAPSHOT_NAME}" -H "Content-Type: application/json" --data '{"manifest_digest": "${RETAG_QUAY_SHA}"}' ; \
	fi

# Add a tag in github for this commit sha
# If RETAG_DRY_RUN is set to `true`, then attempt to query the sha;
# If RETAG_DRY_RUN is set to anything other than `true`, then create a ref tag at that sha using RETAG_SNAPSHOT_NAME
.PHONY: retag/git
## Add a git tag to a repo at the commit sha
retag/git: %git:
	$(call assert-set,GITHUB_TOKEN)
	$(call assert-set,RETAG_GITHUB_SHA)
	$(call assert-set,RETAG_SNAPSHOT_NAME)
	$(call assert-set,RETAG_REPO)
	@if [ "${RETAG_DRY_RUN}" == "true" ]; \
	then $(CURL) --fail --output /dev/null -L -X GET -H "Authorization: token ${GITHUB_TOKEN}" "https://api.github.com/repos/${RETAG_REPO}/git/commits/${RETAG_GITHUB_SHA}"; \
	else $(CURL) -L -X POST -H "Authorization: token ${GITHUB_TOKEN}" "https://api.github.com/repos/${RETAG_REPO}/git/refs" --data '{ "ref": "refs/tags/${RETAG_SNAPSHOT_NAME}", "sha": "${RETAG_GITHUB_SHA}" }'; \
	fi

.PHONY: retag/getquaysha
## Get the quay sha256 out of the manifest digest
retag/getquaysha: %getquaysha:
	@$(shell $(SELF) jq/install > /dev/null)
	@$(CURL) --fail --silent -X GET -H "Authorization: Bearer ${QUAY_TOKEN}" "https://quay.io/api/v1/repository/${PIPELINE_MANIFEST_ORG}/${COMPONENT_NAME}/tag/?onlyActiveTags=true&specificTag=${RETAG_QUAY_COMPONENT_TAG}" | $(JQ) -r .tags[0].manifest_digest
