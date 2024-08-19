FROM registry.ci.openshift.org/stolostron/builder:go1.22-linux AS builder
WORKDIR /go/src/github.com/stolostron/multicloud-operators-foundation
COPY . .
ENV GO_PACKAGE github.com/stolostron/multicloud-operators-foundation

RUN make build --warn-undefined-variables
RUN make build-e2e --warn-undefined-variables

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ENV USER_UID=10001 \
    USER_NAME=acm-foundation

COPY --from=builder /go/src/github.com/stolostron/multicloud-operators-foundation/proxyserver /
COPY --from=builder /go/src/github.com/stolostron/multicloud-operators-foundation/controller /
COPY --from=builder /go/src/github.com/stolostron/multicloud-operators-foundation/webhook /
COPY --from=builder /go/src/github.com/stolostron/multicloud-operators-foundation/agent /
COPY --from=builder /go/src/github.com/stolostron/multicloud-operators-foundation/e2e.test /

RUN microdnf update -y && \
    microdnf clean all

USER ${USER_UID}
