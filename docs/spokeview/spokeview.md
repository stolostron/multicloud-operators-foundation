# SpokeView CRD

This document is to summarise SpokeView CRD. SpokeView is defined to get a specified resource on a certain spoke cluster.

- SpokeView is a namespace-scoped CRD, and the definition is [here](/deploy/dev/hub/resources/crds/view.open-cluster-management.io_spokeviews.yaml).
- SpokeView should be applied in the namespace of a certain spoke cluster, and the usage examples is [here](/examples/spokeview).

## CRD Spec

```yaml
apiVersion: view.open-cluster-management.io/v1beta1
kind: SpokeView
metadata:
  name: <CR name>
  namespace: <namespace of spoke cluster>
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
  - lastHeartbeatTime: "2020-05-09T10:05:17Z"
    lastTransitionTime: "2020-05-09T10:05:17Z"
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
- `result` shows the result retrieved from the spoke cluster.
