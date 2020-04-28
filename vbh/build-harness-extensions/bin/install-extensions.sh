#!/bin/bash
export BUILD_HARNESS_EXTENSIONS_ORG=${1:-open-cluster-management}
export BUILD_HARNESS_EXTENSIONS_PROJECT=${2:-build-harness-extensions}
export BUILD_HARNESS_EXTENSIONS_BRANCH=${3:-master}
export GITHUB_USER=${4}
export GITHUB_TOKEN=${5}
export GITHUB_REPO_SECRET="https://${GITHUB_USER}:${GITHUB_TOKEN}@"
export GITHUB_REPO="github.com/${BUILD_HARNESS_EXTENSIONS_ORG}/${BUILD_HARNESS_EXTENSIONS_PROJECT}.git"

# Note - whatever the extension project's name, we're calling it 'build-harness-extensions'
if [ -d "build-harness-extensions" ]; then
  echo "Removing existing build-harness-extensions"
  rm -rf "build-harness-extensions"
fi

echo "Cloning ${GITHUB_REPO}#${BUILD_HARNESS_EXTENSIONS_BRANCH}..."
# Note - whatever the extension project's name, we're calling it 'build-harness-extensions'
git clone -b $BUILD_HARNESS_EXTENSIONS_BRANCH $GITHUB_REPO_SECRET$GITHUB_REPO build-harness-extensions
