clean-integration-test:
	$(RM) ./integration.test
.PHONY: clean-integration-test

clean: clean-integration-test

test-integration: envtest-setup
	go test -c ./test/integration
	./integration.test -ginkgo.slowSpecThreshold=15 -ginkgo.v -ginkgo.failFast
.PHONY: test-integration
