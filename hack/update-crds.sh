#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace


source "$(dirname "${BASH_SOURCE}")/init.sh"

for f in $CRD_FILES
do
    cp $f ./deploy/foundation/hub/resources/crds
done

