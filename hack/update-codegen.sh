#!/bin/bash

source "$(dirname "${BASH_SOURCE}")/init.sh"

REPO_ROOT=$(realpath $(dirname ${BASH_SOURCE})/..)
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${REPO_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../../../k8s.io/code-generator)}

verify="${VERIFY:-}"

source "${CODEGEN_PKG}/kube_codegen.sh"

# By default, it will generate deepcopy, defaulter and conversion for all types under the pkg/apis directory
kube::codegen::gen_helpers \
    --boilerplate "${REPO_ROOT}/hack/custom-boilerplate.go.txt" \
    ${REPO_ROOT}/pkg/proxyserver/apis/proxy

echo "${REPO_ROOT}/hack/.api_violation.report"

# Generate openapi for servicecatalog and settings group
kube::codegen::gen_openapi \
    --boilerplate "${REPO_ROOT}/hack/custom-boilerplate.go.txt" \
    --update-report \
    --output-dir "${REPO_ROOT}/pkg/proxyserver/apis/openapi" \
    --output-pkg "${REPO_ROOT}/pkg/proxyserver/apis/openapi" \
    --extra-pkgs "open-cluster-management.io/api/cluster/v1" \
    --extra-pkgs "open-cluster-management.io/api/cluster/v1beta2" \
    --extra-pkgs "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1" \
    --extra-pkgs "k8s.io/api/rbac/v1" \
    --extra-pkgs "k8s.io/apimachinery/pkg/api/resource" \
    --extra-pkgs "k8s.io/apimachinery/pkg/runtime" \
    --extra-pkgs "k8s.io/apimachinery/pkg/apis/meta/v1" \
    ${REPO_ROOT}/pkg/proxyserver/apis
