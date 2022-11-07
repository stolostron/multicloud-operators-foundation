#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

CRD_FILES="./vendor/github.com/stolostron/cluster-lifecycle-api/action/v1beta1/action.open-cluster-management.io_managedclusteractions.crd.yaml
./vendor/github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1/internal.open-cluster-management.io_managedclusterinfos.crd.yaml
./vendor/github.com/stolostron/cluster-lifecycle-api/imageregistry/v1alpha1/imageregistry.open-cluster-management.io_managedclusterimageregistries.crd.yaml
./vendor/github.com/stolostron/cluster-lifecycle-api/view/v1beta1/view.open-cluster-management.io_managedclusterviews.crd.yaml
"
