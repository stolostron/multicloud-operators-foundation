test-integration: envtest-setup
	go test -c ./test/integration
	./integration.test -ginkgo.slowSpecThreshold=15 -ginkgo.v -ginkgo.failFast
.PHONY: test-integration
