FROM registry.access.redhat.com/ubi7/ubi-minimal:7.7-98

COPY output/mcm-apiserver output/mcm-webhook output/mcm-controller output/klusterlet output/klusterlet-connectionmanager output/serviceregistry /
