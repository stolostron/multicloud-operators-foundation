#!/bin/bash

source "$(dirname "${BASH_SOURCE}")/init.sh"

for f in $CRD_FILES
do
    diff -N $f ./deploy/foundation/hub/resources/crds/$(basename $f) || ( echo 'crd content is incorrect' && false )
done
