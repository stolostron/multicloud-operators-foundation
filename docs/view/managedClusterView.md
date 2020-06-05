# ManagedClusterView CRD

This document is to summarise ManagedClusterView CRD. ManagedClusterView is defined to get a specified resource on a certain managed cluster.

- ManagedClusterView is a namespace-scoped CRD, and the definition is [here](/deploy/dev/hub/resources/crds/view.open-cluster-management.io_managedclusterviews.yaml).
- ManagedClusterView should be applied in the namespace of a certain managed cluster, and the usage examples is [here](/examples/view).

## CRD Spec

```yaml
apiVersion: view.open-cluster-management.io/v1beta1
kind: ManagedClusterView
metadata:
  name: <CR name>
  namespace: <namespace of managed cluster>
spec:
  scope:
    apiGroup: <optional, group of resource>
    kind: <optional, kind of resource>
    version: <optional, version of resource>
    resource: <optional, type of resource>
    name: <required, name of resource>
    namespace: <optional, namespace of resource>
    updateIntervalSeconds: <optional, the interval to update the resource, default is 30>
status:
  conditions:
  - lastTransitionTime: "2020-05-09T10:05:17Z"
    status: "True"
    type: Processing
    reason: ...
    message: ...
  result:
    <the payload of resource>
```

In the `spec` section:

- `scope.name` is required, and either GKV (`scope.apiGroup+kind+version`) or `scope.resource` should be required too.

In `status` section:

- `conditions` includes only one condition type `Processing`. The status is `True` when it is successful to retrieve resource. Otherwise, the status is `False` if it is fail to retrieve the resource, and the reason is the failure details.
- `result` shows the result retrieved from the managed cluster.
