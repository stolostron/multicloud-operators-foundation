# open-cluster-management Foundation

open-cluster-management Foundation supports some foundational components based ManagedCluster for ACM.

## Community, discussion, contribution, and support

Check the [CONTRIBUTING Doc](CONTRIBUTING.md) for how to contribute to the repo.

------

## Getting Started

This is a guide on how to build and deploy open-cluster-management Foundation from code.

### Build images

Run the following after cloning/pulling/making a change.

```sh
make images
```

`make images` will build a new image named `quay.io/open-cluster-management/multicloud-manager:latest`.

### Prerequisites

Need to install **Cluster Manager** and **Klusterlet** before deploy Foundation components. The installation instruction is [here](https://open-cluster-management.io). 

Need to approve and accept the managed clusters registered to the Hub.
 
* Approve CSR on Hub cluster.

    ```sh
    MANAGED_CLUSTER=$(kubectl get managedclusters | grep cluster | awk '{print $1}')
    CSR_NAME=$(kubectl get csr |grep $MANAGED_CLUSTER | grep Pending |awk '{print $1}')
    kubectl certificate approve "${CSR_NAME}"
    ```

* Accept Managed Cluster on Hub.

    ```sh
    MANAGED_CLUSTER=$(kubectl get managedclusters | grep cluster | awk '{print $1}')
    kubectl patch managedclusters $MANAGED_CLUSTER  --type merge --patch '{"spec":{"hubAcceptsClient":true}}'
    ```

### Deploy Foundation

1. Deploy foundation hub components on hub cluster.

    ```sh
    make deploy-foundation-hub
    ```

2. Deploy foundation agent components to the managed cluster from the hub cluster.
   
    ```sh
    export MANAGED_CLUSTER_NAME=<your managed cluster name, default is cluster1>
    make deploy-foundation-agent
    ```

## Security Response

If you've found a security issue that you'd like to disclose confidentially please contact
Red Hat's Product Security team. Details at [here](https://access.redhat.com/security/team/contact).
