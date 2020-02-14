#!/bin/bash

set -o errexi
set -o nounset
set -o pipefail
set -o xtrace

# Prepare lint tools
# Install hadolint
HADOLINT_PATH="${HOME}"/hadolint
mkdir -p "${HADOLINT_PATH}"
wget -P "${HADOLINT_PATH}" https://github.com/hadolint/hadolint/releases/download/v1.17.5/hadolint-Linux-x86_64
mv "${HADOLINT_PATH}"/hadolint-Linux-x86_64 "${HADOLINT_PATH}"/hadolint
chmod +x "${HADOLINT_PATH}"/hadolint
export PATH="${HADOLINT_PATH}":"${PATH}"

# Install yamllint
pip install --user yamllint

# Install markdown lint
gem install mdl
gem install awesome_bot

# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)"/bin v1.23.6

# Start lint task
make lint