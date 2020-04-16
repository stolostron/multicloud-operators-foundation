# Using BareMetal Inventory Asset

The inventory controller defines a CRD called BareMetalAsset, which is used to hold inventory records for use in baremetal clusters. The controller runs in the hub cluster. The assets are created in a namespace there and the controller will be responsible for reconciling the inventory asset with BareMetalHost resources in the managed cluster.

## Create a new inventory asset

A BareMetalAsset (BMA) represents the hardware available for use in baremetal clusters.

Create a BareMetalAsset in the default (or any) namespace. Each BMA also has a corresponding Secret that contains the BMC credentials and the secret name is referenced by bma.bmc.credentialsName.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: worker-0-bmc-secret
type: Opaque
data:
  username: YWRtaW4=
  password: cGFzc3dvcmQ=
```

```yaml
apiVersion: inventory.open-cluster-management.io/v1alpha1
kind: BareMetalAsset
metadata:
  name: baremetalasset-worker-0
spec:
  bmc:
    address: ipmi://192.168.122.1:6233
    credentialsName: worker-0-bmc-secret
  bootMACAddress: "00:1B:44:11:3A:B7"
  hardwareProfile: "hardwareProfile"
```

A look at the BareMetalAsset status shows that secret referenced was found and the BMA is not associated with any ClusterDeployment yet.

```yaml
status:
  conditions:
  - lastHeartbeatTime: "2020-02-26T19:01:42Z"
    lastTransitionTime: "2020-02-26T19:01:42Z"
    message: A secret with the name worker-0-bmc-secret in namespace default was found
    reason: SecretFound
    status: "True"
    type: CredentialsFound
  - lastHeartbeatTime: "2020-02-26T19:01:42Z"
    lastTransitionTime: "2020-02-26T19:01:42Z"
    message: No cluster deployment specified
    reason: NoneSpecified
    status: "False"
    type: ClusterDeploymentFound
```

Also, note that BMA object metadata has no values populated for the following labels yet.

```yaml
metadata:
  creationTimestamp: "2020-02-26T19:01:42Z"
  finalizers:
  - baremetalasset.inventory.open-cluster-management.io
  generation: 2
  labels:
    metal3.io/cluster-deployment-name: ""
    metal3.io/cluster-deployment-namespace: ""
    metal3.io/role: ""
```

## Add inventory to the cluster

Each BMA can have a role and be associated with a clusterDeployment. Role can either be a "worker" or "master".

Update the BMA spec with role set to worker, and clusterDeployment name and namespace set to the appropriate values of the Hive ClusterDeployment you want the BMA associated with. Eg.

```yaml
  clusterDeployment:
    name: cluster0
    namespace: cluster0
  role: worker
```

You will notice that the metadata labels are updated to their appropriate values. With the labels set, a management application can look for BMAs with a partcular role and clusterDeployment and add them to the cluster.

```yaml
apiVersion: inventory.open-cluster-management.io/v1alpha1
kind: BareMetalAsset
metadata:
  creationTimestamp: "2020-02-26T19:01:42Z"
  finalizers:
  - baremetalasset.inventory.open-cluster-management.io
  generation: 4
  labels:
    metal3.io/cluster-deployment-name: cluster0
    metal3.io/cluster-deployment-namespace: cluster0
    metal3.io/role: worker
  name: baremetalasset-worker-0
  namespace: default
  resourceVersion: "23751"
  selfLink: /apis/inventory.open-cluster-management.io/v1alpha1/namespaces/default/baremetalassets/baremetalasset-worker-0
  uid: dd83e7c1-2882-4aa8-bb6f-a36cb428896c
spec:
  bmc:
    address: ipmi://192.168.122.1:6233
    credentialsName: worker-0-bmc-secret
  bootMACAddress: 00:1B:44:11:3A:B7
  clusterDeployment:
    creationTimestamp: null
    name: cluster0
    namespace: cluster0
  hardwareProfile: hardwareProfile
  role: worker
```

Once an asset is associated with a clusterDeployment, the controller creates a Hive SyncSet for each BMA in the namespace of clusterDeployment. The inventory controller maps the BareMetalAsset to a corresponding BareMetalHost resource in the SyncSet and the SyncSet is responsible for syncing the asset to a BareMetalHost resource in the managed cluster.

Look at BareMetalAsset status conditions to view information about the success or failure of the operations.

```bash
kubectl get baremetalassets baremetalasset-worker-0 -o yaml
```

You can also directly look at SyncSet and SyncSetInstances created by the controller.

```bash
kubectl get syncsets -n cluster0 -o yaml
kubectl get syncsetinstances -n cluster0 -o yaml
```

You can verify the corresponding Secret and BareMetalHost resources are created on the managed cluster in the openshift-machine-api namespace.

```bash
kubectl get secrets worker-0-bmc-secret -n openshift-machine-api
kubectl get baremetalhosts baremetalasset-worker-0 -n openshift-machine-api
```

## Remove inventory from the cluster

To remove an asset from a cluster, you can simply remove or empty the namd and namespace values for clusterDeployment. The SyncSet will be deleted by the controller and BareMetalHost for the asset on the managed cluster will be deprovisioned and removed from the cluster.

```yaml
  clusterDeployment:
    name: ""
    namespace: ""
```

## Update Host information

Any information in the BMA spec like credentials, bootMACAddress, etc. can be updated and the controller will deliver the updates to the managed clusters.