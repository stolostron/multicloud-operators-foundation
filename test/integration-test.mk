TEST_TMP :=/tmp

export KUBEBUILDER_ASSETS ?=$(TEST_TMP)/kubebuilder/bin

K8S_VERSION ?=1.23.1

SETUP_ENVTEST := $(shell go env GOPATH)/bin/setup-envtest

# download the kubebuilder-tools to get kube-apiserver binaries from it
ensure-kubebuilder-tools:
ifeq "" "$(wildcard $(KUBEBUILDER_ASSETS))"
	$(info Downloading kube-apiserver into '$(KUBEBUILDER_ASSETS)')
	mkdir -p '$(KUBEBUILDER_ASSETS)'
ifeq "" "$(wildcard $(SETUP_ENVTEST))"
	$(info Installing setup-envtest into '$(SETUP_ENVTEST)')
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.22
endif
	ENVTEST_K8S_PATH=$$($(SETUP_ENVTEST) use $(K8S_VERSION) --bin-dir $(KUBEBUILDER_ASSETS) -p path); \
	if [ -z "$$ENVTEST_K8S_PATH" ]; then \
		echo "Error: setup-envtest returned empty path"; \
		exit 1; \
	fi; \
	if [ ! -d "$$ENVTEST_K8S_PATH" ]; then \
		echo "Error: setup-envtest path does not exist: $$ENVTEST_K8S_PATH"; \
		exit 1; \
	fi; \
	cp -r $$ENVTEST_K8S_PATH/* $(KUBEBUILDER_ASSETS)/
else
	$(info Using existing kube-apiserver from "$(KUBEBUILDER_ASSETS)")
endif
.PHONY: ensure-kubebuilder-tools

clean-integration-test:
	rm -rf $(TEST_TMP)/kubebuilder
	$(RM) ./integration.test
.PHONY: clean-integration-test

clean: clean-integration-test

test-integration: ensure-kubebuilder-tools
	go test -c ./test/integration
	./integration.test -ginkgo.slowSpecThreshold=15 -ginkgo.v -ginkgo.failFast
.PHONY: test-integration
