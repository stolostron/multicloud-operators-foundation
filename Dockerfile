FROM docker.io/openshift/origin-release:golang-1.13 AS builder
WORKDIR /go/src/github.com/open-cluster-management/multicloud-operators-foundation
COPY . .
ENV GO_PACKAGE github.com/open-cluster-management/multicloud-operators-foundation

RUN make build --warn-undefined-variables

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

COPY --from=builder /go/src/github.com/open-cluster-management/multicloud-operators-foundation/acm-proxyserver /
COPY --from=builder /go/src/github.com/open-cluster-management/multicloud-operators-foundation/acm-controller /
COPY --from=builder /go/src/github.com/open-cluster-management/multicloud-operators-foundation/acm-webhook /
COPY --from=builder /go/src/github.com/open-cluster-management/multicloud-operators-foundation/acm-agent /

RUN microdnf update && \
    microdnf clean all
