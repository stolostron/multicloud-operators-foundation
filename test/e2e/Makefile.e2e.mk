DATETIME = $(shell date +'%Y%m%d%H%M%S')
LOGFILE = "/tmp/multicloud-manager-e2e-test-${DATETIME}.log"

.PHONY: e2e-test-check-default-kubefile e2e-test-check-kubefile e2e-test-install-ginkgo

e2e-test-check-default-kubefile:
ifeq ($(wildcard  ~/.kube/config),)
	@$(info  Default config not exists. Please set env variable: KUBECONFIG)
	@exit -1
else
	@exit 0
endif

e2e-test-check-kubefile:
ifeq ($(wildcard  ${KUBECONFIG}),)
	@$(info  File [${KUBECONFIG}] not exists)
	@exit -1
else
	@exit 0
endif

e2e-test-install-ginkgo:
	@go get github.com/onsi/ginkgo/ginkgo

.PHONY: init-e2e-test run-e2e-test run-all-e2e-test

init-e2e-test:
ifndef KUBECONFIG
	@make e2e-test-check-default-kubefile
else
	@make e2e-test-check-kubefile
endif
	@make e2e-test-install-ginkgo

run-e2e-test: init-e2e-test
ifndef TEST_SUITE
	@$(info TEST_SUITE not defined)
	@exit -1
endif
ifneq ($(wildcard test/e2e/tests/${TEST_SUITE}),)
	@ginkgo -v -tags integration --slowSpecThreshold=15 --failFast test/e2e/tests/${TEST_SUITE}/... 
else
	$(info Test suite [${TEST_SUITE}] not exists)
	@exit -1
endif

run-all-e2e-test: init-e2e-test
#	@ginkgo -v -tags integration --slowSpecThreshold=15 --failFast test/e2e/tests/... 
	@ginkgo -v -tags integration --slowSpecThreshold=15 --failFast test/e2e/tests/clusterinfo/...
	@ginkgo -v -tags integration --slowSpecThreshold=15 --failFast test/e2e/tests/views/...
	@ginkgo -v -tags integration --slowSpecThreshold=15 --failFast test/e2e/tests/actions/...
	@ginkgo -v -tags integration --slowSpecThreshold=15 --failFast test/e2e/tests/cluster/...
