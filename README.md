# ACM Foundation

ACM Foundation supports some foundational components based ManagedCluster for ACM.

## Community, discussion, contribution, and support

Check the [CONTRIBUTING Doc](CONTRIBUTING.md) for how to contribute to the repo.

------

## Getting Started

This is a guide on how to build and deploy ACM Foundation from code.

### Setup

Create a directory `$GOPATH/src/github.com/open-cluster-management`, and clone the code into the directory. Since the build process will use (eventually installing) some `golang` tools, makes sure you have `$GOPATH/bin` added to your `$PATH`. 

Populate the vendor directory. If necessary, set environment variable `GO111MODULE=on`.

```sh
go mod vendor
```

### Build

Run the following after cloning/pulling/making a change.

```sh
make build
```

make build will build all the binaries in the current directory.

### Prerequisites

Need to install ManagedCluster before deploy ACM Foundation.

1. Install Cluster Manager on Hub cluster.

    ```sh
    make deploy-hub
    ```

2. Install Klusterlet On Managed cluster.

    1. Copy `kubeconfig` of Hub to `~/.kubconfig`.
    2. Install Klusterlet.

        ```sh
        make deploy-klusterlet
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

### Deploy ACM Foundation from the quay.io

1. Deploy hub components on hub cluster.

    ```sh
    make deploy-foundation-hub
    ```

2. Deploy klusterlet components on managed cluster.

    ```sh
    make deploy-foundation-agent
    ```

## Security Response

If you've found a security issue that you'd like to disclose confidentially please contact
Red Hat's Product Security team. Details at [here](https://access.redhat.com/security/team/contact).
