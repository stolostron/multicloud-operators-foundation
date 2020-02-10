FROM registry.access.redhat.com/ubi7/ubi-minimal:7.7-98

ENV USER_UID=10001 \
    USER_NAME=mcm-operator

COPY output/mcm-apiserver output/mcm-webhook output/mcm-controller output/klusterlet output/klusterlet-connectionmanager output/serviceregistry /
COPY build/bin /usr/local/bin

RUN /usr/local/bin/user_setup

RUN microdnf update && \
    microdnf clean all

USER ${USER_UID}
