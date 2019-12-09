#!/bin/bash

# licensed Materials - Property of IBM
# 5737-E67
# (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
# US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

# Release Tag
# DOCKER_TAG is used by build-harness
if [ "$TRAVIS_BRANCH" = "master" ]; then
    RELEASE_TAG=latest
    DOCKER_TAG=latest
else
    RELEASE_TAG="${TRAVIS_BRANCH#release-}-latest"
    DOCKER_TAG="${TRAVIS_BRANCH#release-}-latest"
fi
if [ "$TRAVIS_TAG" != "" ]; then
    RELEASE_TAG="${TRAVIS_TAG#v}"
    DOCKER_TAG="${TRAVIS_TAG#v}"
fi
export RELEASE_TAG="$RELEASE_TAG"
export DOCKER_TAG="$DOCKER_TAG"

# Release Tag
echo TRAVIS_EVENT_TYPE="$TRAVIS_EVENT_TYPE"
echo TRAVIS_BRANCH="$TRAVIS_BRANCH"
echo TRAVIS_TAG="$TRAVIS_TAG"
echo RELEASE_TAG="$RELEASE_TAG"
echo DOCKER_TAG="$DOCKER_TAG"
