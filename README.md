<p align="center"><a href="http://35.227.205.240/?job=build-multicloud-operators-foundation_postsubmit">
<img alt="Build Status" src="http://35.227.205.240/badge.svg?jobs=build-multicloud-operators-foundation_postsubmit">
</a>
</p>

# ACM Foundation

ACM Foundation supports some foundational components based ManagedCluster for ACM.

This is a guide on how to build and deploy ACM Foundation from code.

## Setup

Create a directory `$GOPATH/src/github.com/open-cluster-management`, and clone the code into the directory.

Populate the vendor directory. If necessary, set environment variable `GO111MODULE=on`.

```sh
go mod vendor
```

## Build

Run the following after cloning/pulling/making a change.

```sh
make build
```

make build will build all the binaries in output directory.

## Prerequisites

Need to install ManagedCluster before deploy ACM Foundation.

1. Install Cluster Mananger on Hub cluster.

    ```sh
    bash deploy/managedcluster/hub/install.sh
    ```

2. Install Klusterlet On Managed cluster.

    1. Copy `kubeconfig` of Hub to `~/.kubconfig`.
    2. Install Klusterlet.

        ```sh
        bash deploy/managedcluster/klusterlet/install.sh
        ```

3. Approve CSR on Hub cluster.

    ```sh
    MANAGED_CLUSTER=$(kubectl get managedclusters | grep cluster | awk '{print $1}')
    CSR_NAME=$(kubectl get csr |grep $MANAGED_CLUSTER | grep Pending |awk '{print $1}')
    kubectl certificate approve "${CSR_NAME}"
    ```

4. Accept Managed Cluster on Hub.

    ```sh
    MANAGED_CLUSTER=$(kubectl get managedclusters | grep cluster | awk '{print $1}')
    kubectl patch managedclusters $MANAGED_CLUSTER  --type merge --patch '{"spec":{"hubAcceptsClient":true}}'
    ```

## Deploy ACM Foundation from quay.io

You can use `kustomize` to deploy ACM Foundation with the following step.

Install `kustomize`, you can use

```sh
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash mv kustomize /usr/local/bin/
```

More info please see: [kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md)

1. Install on hub cluster

    Config image repo and image pull secret in `deploy/prod/hub/kustomization.yaml`

    - Change image repo and image tag in `images` section
    - Change `<AUTH INFO>` with your image repo login info in `secretGenerator` section. It looks like:

        ```json
        {"auths":{"https://quay.io/open-cluster-management/multicloud-manager":{"username":"<USER NAME>","password":"<TOKEN>/","auth":"<BASE64 ENACODE <USER:TOKEN>>"}}}
        ```

    Deploy hub components

    ```sh
    kustomize build deploy/prod/hub | kubectl apply  -f -
    ```

2. Install on managed cluster

    Config image repo and image pull secret info in `deploy/prod/klusterlet/kustomization.yaml`

    - Change image repo and image tag in `images` section
    - Change `<AUTH INFO>` with your image repo login info in `secretGenerator` section. It looks like:

        ```json
        {"auths":{"https://quay.io/open-cluster-management/multicloud-manager":{"username":"<USER NAME>","password":"<TOKEN>/","auth":"<BASE64 ENACODE <USER:TOKEN>>"}}}
        ```

    Deploy klusterlet components

    ```sh
    kustomize build deploy/prod/klusterlet | kubectl apply  -f -
    ```

## Deploy for development environment

You can use `ko` to deploy ACM Foundation with the following step.

Install `ko`, you can use

```sh
go get github.com/google/ko/cmd/ko
```

More information see [ko](https://github.com/google/ko)

> Note:
> * Go version needs >= go1.11.
> * Need `export GO111MODULE=on` if Go version is go1.11 or go1.12.

1. Config `KO_DOCKER_REPO` for deployment tool **ko**

    Configure `KO_DOCKER_REPO` by running `gcloud auth configure-docker` if you are using Google Container Registry or `docker login` if you are using Docker Hub.

    ```sh
    export PROJECT_ID=$(gcloud config get-value core/project)
    export KO_DOCKER_REPO="gcr.io/${PROJECT_ID}"
    ```

    or

    ```sh
    export KO_DOCKER_REPO=docker.io/<your account>
    ```

2. Install on hub cluster

    Deploy hub components

    ```sh
    ko apply -f deploy/dev/hub/resources --base-import-paths --tags=latest
    ```

3. Install klusterlet-addon on the hub cluster

    Deploy klusterlet components

    ```sh
    ko apply -f deploy/dev/klusterlet/resources --base-import-paths --tags=latest
    ```
