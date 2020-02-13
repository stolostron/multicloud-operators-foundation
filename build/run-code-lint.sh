#!/bin/bash

set -o errexi
set -o nounset
set -o pipefail
set -o xtrace

# prepare lint tools
LINT_TOOLS_PATH="${HOME}"/lint-tools

mkdir -p "${LINT_TOOLS_PATH}"

wget -P "${LINT_TOOLS_PATH}" https://github.com/hadolint/hadolint/releases/download/v1.17.5/hadolint-Linux-x86_64
mv "${LINT_TOOLS_PATH}"/hadolint-Linux-x86_64 "${LINT_TOOLS_PATH}"/hadolint
chmod +x "${LINT_TOOLS_PATH}"/hadolint

pip install --user yamllint

export PATH="${LINT_TOOLS_PATH}":"${PATH}"

# start lint ...
make lint