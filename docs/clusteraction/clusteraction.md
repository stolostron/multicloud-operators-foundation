# ClusterAction CRD

This document is to summarise ClusterAction CRD. ClusterAction is defined as a certain action job executed on a certain spoke cluster to Create/Update/Delete a resource.

- ClusterAction is a namespace-scoped CRD, and the definition is [here](/deploy/dev/hub/resources/crds/action.open-cluster-management.io_clusteractions.yaml).
- ClusterAction should be applied in the namespace of a certain spoke cluster, and the usage examples is [here](/examples/action).

## CRD Spec

```yaml
apiVersion: action.open-cluster-management.io/v1beta1
kind: ClusterAction
metadata:
  name: <CR name>
  namespace: <namespace of spoke cluster>
spec:
  actionType: <Create/Update/Delete>
  kube:
    <kube resource payload>
status:
  conditions:
  - lastHeartbeatTime: "2020-05-09T10:05:17Z"
    lastTransitionTime: "2020-05-09T10:05:17Z"
    status: "False"
    type: Completed
    reason: ...
    message: ...
  result:
    <references the related result of the action>
```

In the `spec` section:

- `actionType` is the type of action which can be `Create`,`Update` or `Delete`.
- `kube` is the payload of reousrce to process.

In `status` section:

- `conditions` includes only one condition type `Completed`. The status is `True` if execute the action successfully. Otherwise, the status is `False` if action fails, and the reason is the failure details.
- `result` show the related result of the action.


