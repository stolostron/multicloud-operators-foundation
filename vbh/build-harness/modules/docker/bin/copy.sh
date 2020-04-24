#!/bin/bash

export FROM_DOCKER_USER=${1:-}
export FROM_DOCKER_PASS=${2:-}
export FROM_DOCKER_REGISTRY=${3:-}
export FROM_DOCKER_NAMESPACE=${4:-}
export FROM_DOCKER_IMAGE=${5:-}
export FROM_DOCKER_TAG=${6:-}
export TO_DOCKER_USER=${7:-}
export TO_DOCKER_PASS=${8:-}
export TO_DOCKER_REGISTRY=${9:-}
export TO_DOCKER_NAMESPACE=${10:-}
export TO_DOCKER_IMAGE=${11:-}
export TO_DOCKER_TAG=${12:-}
export ADDTL_TAG=${13:-}

echo "login to FROM_DOCKER_REGISTRY: $FROM_DOCKER_REGISTRY"
NEXT_WAIT_TIME=0
COMMAND_STATUS=1
until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
    echo "                                                                              "
    echo "attempt #$NEXT_WAIT_TIME:"
    docker login -u $FROM_DOCKER_USER -p $FROM_DOCKER_PASS $FROM_DOCKER_REGISTRY
    COMMAND_STATUS=$?
    sleep $(( NEXT_WAIT_TIME++ ))
done
echo "login to TO_DOCKER_REGISTRY: $TO_DOCKER_REGISTRY"
NEXT_WAIT_TIME=0
COMMAND_STATUS=1
until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
    echo "                                                                              "
    echo "attempt #$NEXT_WAIT_TIME:"
    docker login -u $TO_DOCKER_USER -p $TO_DOCKER_PASS $TO_DOCKER_REGISTRY
    COMMAND_STATUS=$?
    sleep $(( NEXT_WAIT_TIME++ ))
done

# sudo chown -R $USER: ~/.docker
# sudo chmod -R g+rwx ~/.docker
# jq '.experimental = "enabled"' ~/.docker/config.json | tee ~/.docker/config.json

echo "                                                                              "
echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
echo "Start time: $(date "+%Y%m%d-%H:%M:%S")"
echo "Move $FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG"
echo "     from: $FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/$FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG"
echo "       to: $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}:$TO_DOCKER_TAG"
echo "                                                                              "
echo "docker manifest inspect $FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/$FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG | jq '.mediaType == \"application/vnd.docker.distribution.manifest.list.v2+json\"'"
echo "pulling manifest for $FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/$FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG"
NEXT_WAIT_TIME=0
COMMAND_STATUS=1
until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
    echo "                                                                              "
    echo "attempt #$NEXT_WAIT_TIME:"
    docker manifest inspect $FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/$FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG > /tmp/$FROM_DOCKER_IMAGE-manifest.json
    COMMAND_STATUS=$?
    sleep $(( NEXT_WAIT_TIME++ ))
done
export IS_MULTIARCH=$(cat /tmp/$FROM_DOCKER_IMAGE-manifest.json | jq ".mediaType == \"application/vnd.docker.distribution.manifest.list.v2+json\"")

if [ $IS_MULTIARCH == "true" ]; then

    echo "Found multi-arch image..."

	# gather arch digests
	AMD64_DIGEST=$(cat /tmp/$FROM_DOCKER_IMAGE-manifest.json  | jq ".manifests[] | select(.platform.architecture == \"amd64\") | .digest" | sed 's/\"//g')
	PPC64LE_DIGEST=$(cat /tmp/$FROM_DOCKER_IMAGE-manifest.json  | jq ".manifests[] | select(.platform.architecture == \"ppc64le\") | .digest" | sed 's/\"//g')
	S390X_DIGEST=$(cat /tmp/$FROM_DOCKER_IMAGE-manifest.json  | jq ".manifests[] | select(.platform.architecture == \"s390x\") | .digest" | sed 's/\"//g')

    echo "                                                                              "
    echo "amd64 digest: $AMD64_DIGEST"
    echo "ppc64le digest: $PPC64LE_DIGEST"
    echo "s390x digest: $S390X_DIGEST"
    echo "                                                                              "
    echo "                                                                              "

	# pull digest images
    if ! [ -z "$AMD64_DIGEST" ]; then
        IMG_URI="${FROM_DOCKER_REGISTRY}/${FROM_DOCKER_NAMESPACE}/${FROM_DOCKER_IMAGE}@${AMD64_DIGEST}"
        echo "                                                                              "
        echo "pulling amd64 digest: docker pull $IMG_URI"
        echo "                                                                              "
        NEXT_WAIT_TIME=0
        COMMAND_STATUS=1
        until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
            echo "                                                                              "
            echo "attempt #$NEXT_WAIT_TIME:"
            docker pull $IMG_URI
            COMMAND_STATUS=$?
            sleep $(( NEXT_WAIT_TIME++ ))
        done
    fi
    if ! [ -z "$PPC64LE_DIGEST" ]; then
        IMG_URI="${FROM_DOCKER_REGISTRY}/${FROM_DOCKER_NAMESPACE}/${FROM_DOCKER_IMAGE}@${PPC64LE_DIGEST}"
        echo "                                                                              "
        echo "pulling ppc64le digest: docker pull $IMG_URI"
        echo "                                                                              "
        NEXT_WAIT_TIME=0
        COMMAND_STATUS=1
        until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
            echo "                                                                              "
            echo "attempt #$NEXT_WAIT_TIME:"
            docker pull $IMG_URI
            COMMAND_STATUS=$?
            sleep $(( NEXT_WAIT_TIME++ ))
        done
    fi
    if ! [ -z "$S390X_DIGEST" ]; then
        IMG_URI="${FROM_DOCKER_REGISTRY}/${FROM_DOCKER_NAMESPACE}/${FROM_DOCKER_IMAGE}@${S390X_DIGEST}"
        echo "                                                                              "
        echo "pulling s390x digest: docker pull $IMG_URI"
        echo "                                                                              "
        NEXT_WAIT_TIME=0
        COMMAND_STATUS=1
        until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
            echo "                                                                              "
            echo "attempt #$NEXT_WAIT_TIME:"
            docker pull $IMG_URI
            COMMAND_STATUS=$?
            sleep $(( NEXT_WAIT_TIME++ ))
        done
    fi

	# tag digests for destination (as well as tagging date)
    if ! [ -z "$AMD64_DIGEST" ]; then
        echo "                                                                              "
        echo "tagging amd64 digest"
	    docker tag "${FROM_DOCKER_REGISTRY}/${FROM_DOCKER_NAMESPACE}/${FROM_DOCKER_IMAGE}@${AMD64_DIGEST}" $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-amd64:$TO_DOCKER_TAG
	    docker tag "${FROM_DOCKER_REGISTRY}/${FROM_DOCKER_NAMESPACE}/${FROM_DOCKER_IMAGE}@${AMD64_DIGEST}" $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-amd64:$ADDTL_TAG
    fi
    if ! [ -z "$PPC64LE_DIGEST" ]; then
        echo "                                                                              "
        echo "tagging ppc64le digest"
	    docker tag "${FROM_DOCKER_REGISTRY}/${FROM_DOCKER_NAMESPACE}/${FROM_DOCKER_IMAGE}@${PPC64LE_DIGEST}" $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-ppc64le:$TO_DOCKER_TAG
	    docker tag "${FROM_DOCKER_REGISTRY}/${FROM_DOCKER_NAMESPACE}/${FROM_DOCKER_IMAGE}@${PPC64LE_DIGEST}" $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-ppc64le:$ADDTL_TAG
    fi
    if ! [ -z "$S390X_DIGEST" ]; then
        echo "                                                                              "
        echo "tagging s390x digest"
	    docker tag "${FROM_DOCKER_REGISTRY}/${FROM_DOCKER_NAMESPACE}/${FROM_DOCKER_IMAGE}@${S390X_DIGEST}" $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-s390x:$TO_DOCKER_TAG
	    docker tag "${FROM_DOCKER_REGISTRY}/${FROM_DOCKER_NAMESPACE}/${FROM_DOCKER_IMAGE}@${S390X_DIGEST}" $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-s390x:$ADDTL_TAG
    fi

	# push tagged digests into destination
    if ! [ -z "$AMD64_DIGEST" ]; then
        echo "                                                                              "
        echo "pushing amd64 digest: docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-amd64:$TO_DOCKER_TAG"
        echo "                                                                              "
        NEXT_WAIT_TIME=0
        COMMAND_STATUS=1
        until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
            echo "                                                                              "
            echo "attempt #$NEXT_WAIT_TIME:"
            docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-amd64:$TO_DOCKER_TAG
            COMMAND_STATUS=$?
            sleep $(( NEXT_WAIT_TIME++ ))
        done
        echo "                                                                              "
        echo "pushing amd64 digest: docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-amd64:$ADDTL_TAG"
        echo "                                                                              "
        NEXT_WAIT_TIME=0
        COMMAND_STATUS=1
        until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
            echo "                                                                              "
            echo "attempt #$NEXT_WAIT_TIME:"
            docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-amd64:$ADDTL_TAG
            COMMAND_STATUS=$?
            sleep $(( NEXT_WAIT_TIME++ ))
        done
    fi
    if ! [ -z "$PPC64LE_DIGEST" ]; then
        echo "                                                                              "
        echo "pushing ppc64le digest: docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-ppc64le:$TO_DOCKER_TAG"
        echo "                                                                              "
        NEXT_WAIT_TIME=0
        COMMAND_STATUS=1
        until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
            echo "                                                                              "
            echo "attempt #$NEXT_WAIT_TIME:"
            docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-ppc64le:$TO_DOCKER_TAG
            COMMAND_STATUS=$?
            sleep $(( NEXT_WAIT_TIME++ ))
        done
        echo "                                                                              "
        echo "pushing ppc64le digest: docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-ppc64le:$ADDTL_TAG"
        echo "                                                                              "
        NEXT_WAIT_TIME=0
        COMMAND_STATUS=1
        until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
            echo "                                                                              "
            echo "attempt #$NEXT_WAIT_TIME:"
            docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-ppc64le:$ADDTL_TAG
            COMMAND_STATUS=$?
            sleep $(( NEXT_WAIT_TIME++ ))
        done
    fi
    if ! [ -z "$S390X_DIGEST" ]; then
        echo "                                                                              "
        echo "pushing s390x digest: docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-s390x:$TO_DOCKER_TAG"
        echo "                                                                              "
        NEXT_WAIT_TIME=0
        COMMAND_STATUS=1
        until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
            echo "                                                                              "
            echo "attempt #$NEXT_WAIT_TIME:"
            docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-s390x:$TO_DOCKER_TAG
            COMMAND_STATUS=$?
            sleep $(( NEXT_WAIT_TIME++ ))
        done
        echo "                                                                              "
        echo "pushing s390x digest: docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-s390x:$ADDTL_TAG"
        echo "                                                                              "
        NEXT_WAIT_TIME=0
        COMMAND_STATUS=1
        until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
            echo "                                                                              "
            echo "attempt #$NEXT_WAIT_TIME:"
            docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-s390x:$ADDTL_TAG
            COMMAND_STATUS=$?
            sleep $(( NEXT_WAIT_TIME++ ))
        done
    fi

    # if amd64, ppc64le, and s390x digests exist
    if [[ ! -z "$AMD64_DIGEST" && ! -z "$PPC64LE_DIGEST" && ! -z "$S390X_DIGEST" ]]; then
        cat > /tmp/${TO_DOCKER_IMAGE}-manifest.yaml <<EOL
image: __DOCKER_REGISTRY__/__DOCKER_NAMESPACE__/__DOCKER_IMAGE__:__DOCKER_TAG__
tags: ['__ADDTL_TAG__']
manifests:
  - image: __DOCKER_REGISTRY__/__DOCKER_NAMESPACE__/__DOCKER_IMAGE__-amd64:__DOCKER_TAG__
    platform:
      architecture: amd64
      os: linux
  - image: __DOCKER_REGISTRY__/__DOCKER_NAMESPACE__/__DOCKER_IMAGE__-ppc64le:__DOCKER_TAG__
    platform:
      architecture: ppc64le
      os: linux
  - image: __DOCKER_REGISTRY__/__DOCKER_NAMESPACE__/__DOCKER_IMAGE__-s390x:__DOCKER_TAG__
    platform:
      architecture: s390x
      os: linux
EOL
    # if amd64, and ppc64le exist and s390x does not exists
    elif [[ ! -z "$AMD64_DIGEST" && ! -z "$PPC64LE_DIGEST" && -z "$S390X_DIGEST" ]]; then
        cat > /tmp/${TO_DOCKER_IMAGE}-manifest.yaml <<EOL
image: __DOCKER_REGISTRY__/__DOCKER_NAMESPACE__/__DOCKER_IMAGE__:__DOCKER_TAG__
tags: ['__ADDTL_TAG__']
manifests:
  - image: __DOCKER_REGISTRY__/__DOCKER_NAMESPACE__/__DOCKER_IMAGE__-amd64:__DOCKER_TAG__
    platform:
      architecture: amd64
      os: linux
  - image: __DOCKER_REGISTRY__/__DOCKER_NAMESPACE__/__DOCKER_IMAGE__-ppc64le:__DOCKER_TAG__
    platform:
      architecture: ppc64le
      os: linux
EOL
    # if amd64 exists and ppc64le and s390x do not exist
    elif [[ ! -z "$AMD64_DIGEST" && -z "$PPC64LE_DIGEST" && -z "$S390X_DIGEST" ]]; then
        cat > /tmp/${TO_DOCKER_IMAGE}-manifest.yaml <<EOL
image: __DOCKER_REGISTRY__/__DOCKER_NAMESPACE__/__DOCKER_IMAGE__:__DOCKER_TAG__
tags: ['__ADDTL_TAG__']
manifests:
  - image: __DOCKER_REGISTRY__/__DOCKER_NAMESPACE__/__DOCKER_IMAGE__-amd64:__DOCKER_TAG__
    platform:
      architecture: amd64
      os: linux
EOL
    fi

    sed -i.bak -e "s|__DOCKER_REGISTRY__|${TO_DOCKER_REGISTRY}|g" \
               -e "s|__DOCKER_NAMESPACE__|${TO_DOCKER_NAMESPACE}|g" \
               -e "s|__DOCKER_IMAGE__|${TO_DOCKER_IMAGE}|g" \
               -e "s|__DOCKER_TAG__|${TO_DOCKER_TAG}|g" \
               -e "s|__ADDTL_TAG__|${ADDTL_TAG}|g" \
               /tmp/${TO_DOCKER_IMAGE}-manifest.yaml

    echo "                                                                              "
    echo "creating manifest-list image: $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}:$TO_DOCKER_TAG"
    echo "                                                                              "
    NEXT_WAIT_TIME=0
    COMMAND_STATUS=1
    until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
        echo "                                                                              "
        echo "attempt #$NEXT_WAIT_TIME:"
        manifest-tool --debug --username ${TO_DOCKER_USER} --password "${TO_DOCKER_PASS}" push from-spec /tmp/${TO_DOCKER_IMAGE}-manifest.yaml
        COMMAND_STATUS=$?
        sleep $(( NEXT_WAIT_TIME++ ))
    done

    # clean up... delete local images
    if ! [ -z "$AMD64_DIGEST" ]; then
        echo "                                                                              "
        echo "cleaning up amd64 local images..."
        echo "                                                                              "
	    docker rmi -f $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-amd64:$TO_DOCKER_TAG
	    docker rmi -f $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-amd64:$ADDTL_TAG
        docker rmi -f "$FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/${FROM_DOCKER_IMAGE}@$AMD64_DIGEST"
    fi
    if ! [ -z "$PPC64LE_DIGEST" ]; then
        echo "                                                                              "
        echo "cleaning up ppc64le local images..."
        echo "                                                                              "
	    docker rmi -f $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-ppc64le:$TO_DOCKER_TAG
	    docker rmi -f $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-ppc64le:$ADDTL_TAG
        docker rmi -f "$FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/${FROM_DOCKER_IMAGE}@$PPC64LE_DIGEST"
    fi
    if ! [ -z "$S390X_DIGEST" ]; then
        echo "                                                                              "
        echo "cleaning up s390x local images..."
        echo "                                                                              "
	    docker rmi -f $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-s390x:$TO_DOCKER_TAG
	    docker rmi -f $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/${TO_DOCKER_IMAGE}-s390x:$ADDTL_TAG
        docker rmi -f "$FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/${FROM_DOCKER_IMAGE}@$S390X_DIGEST"
    fi
else

    echo "Found non-multi-arch image..."

	# pull docker image
    echo "                                                                              "
    echo "pulling image: docker pull $FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/$FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG"
    echo "                                                                              "
    NEXT_WAIT_TIME=0
    COMMAND_STATUS=1
    until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
        echo "                                                                              "
        echo "attempt #$NEXT_WAIT_TIME:"
        docker pull $FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/$FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG
        COMMAND_STATUS=$?
        sleep $(( NEXT_WAIT_TIME++ ))
    done

	# tag image for destination (as well as tagging date)
    echo "                                                                              "
    echo "tagging docker image with $TO_DOCKER_IMAGE:$TO_DOCKER_TAG"
	docker tag $FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/$FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/$TO_DOCKER_IMAGE:$TO_DOCKER_TAG
    echo "                                                                              "
    echo "tagging docker image with $TO_DOCKER_IMAGE:$ADDTL_TAG"
	docker tag $FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/$FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/$TO_DOCKER_IMAGE:$ADDTL_TAG

	# push image to desitnation
    echo "                                                                              "
    echo "pushing image: docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/$TO_DOCKER_IMAGE:$TO_DOCKER_TAG"
    echo "                                                                              "
    NEXT_WAIT_TIME=0
    COMMAND_STATUS=1
    until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
        echo "                                                                              "
        echo "attempt #$NEXT_WAIT_TIME:"
        docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/$TO_DOCKER_IMAGE:$TO_DOCKER_TAG
        COMMAND_STATUS=$?
        sleep $(( NEXT_WAIT_TIME++ ))
    done
    echo "                                                                              "
    echo "pushing image: docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/$TO_DOCKER_IMAGE:$ADDTL_TAG"
    echo "                                                                              "
    NEXT_WAIT_TIME=0
    COMMAND_STATUS=1
    until [ $COMMAND_STATUS -eq 0 ] || [ $NEXT_WAIT_TIME -eq 4 ]; do
        echo "                                                                              "
        echo "attempt #$NEXT_WAIT_TIME:"
        docker push $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/$TO_DOCKER_IMAGE:$ADDTL_TAG
        COMMAND_STATUS=$?
        sleep $(( NEXT_WAIT_TIME++ ))
    done

    # clean up... delete local images
    echo "                                                                              "
    echo "cleaning up local images..."
    echo "                                                                              "
    docker rmi -f $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/$TO_DOCKER_IMAGE:$ADDTL_TAG
    docker rmi -f $TO_DOCKER_REGISTRY/$TO_DOCKER_NAMESPACE/$TO_DOCKER_IMAGE:$TO_DOCKER_TAG
	docker rmi -f $FROM_DOCKER_REGISTRY/$FROM_DOCKER_NAMESPACE/$FROM_DOCKER_IMAGE:$FROM_DOCKER_TAG
fi
echo "End time: $(date "+%Y%m%d-%H:%M:%S")"
echo "<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"