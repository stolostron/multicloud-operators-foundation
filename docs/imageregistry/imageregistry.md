# ManagedClusterImageRegistry CRD

This document is to summarise ManagedClusterImageRegistry CRD. ManagedClusterImageRegistry is defined as a configuration
to override the images of pods deployed on the managed clusters.

ManagedClusterImageRegistry is a namespace-scoped CRD, and the definition is [here](deploy/foundation/hub/resources/crds/imageregistry.open-cluster-management.io_managedclusterimageregistries.yaml).

ManagedClusterImageRegistry refers a Placement to select a set of ManagedClusters which need to override images from custom image registry.

The selected ManagedClusters will be added a label `open-cluster-management.io/image-registry=<namespace>.<managedClusterImageRegistryName>`.

## CRD Spec

```yaml
apiVersion: imageregistry.open-cluster-management.io/v1alpha1
kind: ManagedClusterImageRegistry
metadata:
  name: <imageRegistryName>
  namespace: <namespace>
spec:
  placementRef:
    group: cluster.open-cluster-management.io
    resource: placements
    name: <placementName> 
  pullSecret:
    name: <pullSecretName>
  registry: <registryAddress>
```
In the `spec` section:

- `placementRef` refers a placement in the same namespace to select a set of managed clusters.
- `pullSecret` is the name of pullSecret used to pull images from the custom image registry.
- `registry` is the custom registry address.


## Example

How to import a cluster with ManagedClusterImageRegistry

1. Create a pullSecret in namespace `myNamespace`.

```bash
$ kubectl create secret docker-registry myPullSecret \
  --docker-server=<your-registry-server> \
  --docker-username=<my-name> \
  --docker-password=<my-password>
```

2. Create a Placement in namespace `myNamespace`.

```yaml
apiVersion: cluster.open-cluster-management.io/v1alpha1
kind: Placement
metadata:
  name: myPlacement
  namespace: myNamespace
```

3. Create a ManagedClusterSet and bind it to the namespace `myNamespace`.

```yaml
apiVersion: cluster.open-cluster-management.io/v1alpha1
kind: ManagedClusterSet
metadata:
  name: myClusterSet
  
---
apiVersion: cluster.open-cluster-management.io/v1alpha1
kind: ManagedClusterSetBinding
metadata:
  name: myClusterSet
  namespace: myNamespace
spec:
  clusterSet: myClusterSet
```

4. Create the ManagedClusterImageRegistry in namespace  `myNamespace`.

```yaml
apiVersion: imageregistry.open-cluster-management.io/v1alpha1
kind: ManagedClusterImageRegistry
metadata:
  name: myImageRegistry
  namespace: myNamespace
spec:
  placementRef:
    group: cluster.open-cluster-management.io
    resource: placements
    name: myPlacement
  pullSecret:
    name: myPullSecret
  registry: myRegistryAddress
```

5. Import a managed cluster from ACM console and add it into ManagedClusterSet `myClusterSet`.
6. Copy and run the import commands on managed cluster after the label `open-cluster-management.io/image-registry=myNamespace.myImageRegistry` is added to the ManagedCluster.
