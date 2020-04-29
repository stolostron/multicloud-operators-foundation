FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1-398

ENV USER_UID=10001 \
    USER_NAME=mcm-operator

COPY output/acm-proxyserver output/acm-agent output/mcm-apiserver output/mcm-webhook output/mcm-controller output/klusterlet output/klusterlet-connectionmanager output/serviceregistry output/acm-controller /

COPY build/bin /usr/local/bin

RUN /usr/local/bin/user_setup

RUN microdnf update && \
    microdnf clean all

USER ${USER_UID}
