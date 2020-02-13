#!/bin/bash

set -o errexi
set -o nounset
set -o pipefail
set -o xtrace

# Install lint tools
LINT_TOOLS_PATH="${HOME}"/lint-tools
mkdir -p "${LINT_TOOLS_PATH}"

export PATH=$LINT_TOOLS_PATH:$PATH

wget -P "${LINT_TOOLS_PATH}"/hadolint https://github.com/hadolint/hadolint/releases/download/v1.17.5/hadolint-Linux-x86_64
chmod +x "${LINT_TOOLS_PATH}"/hadolint
