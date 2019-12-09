FROM docker.io/openshift/origin-release:golang-1.13 AS builder
WORKDIR /go/src/github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/
COPY . .
RUN go mod vendor
RUN make build
RUN mv output /

FROM registry.access.redhat.com/ubi7/ubi-minimal:7.7-98
COPY --from=builder output/mcm-apiserver output/mcm-webhook output/mcm-controller output/klusterlet output/klusterlet-connectionmanager output/serviceregistry /
