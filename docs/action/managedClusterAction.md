# ManagedClusterAction CRD

This document is to summarise ManagedClusterAction CRD. ManagedClusterAction is defined as a certain action job executed on a certain managed cluster to Create/Update/Delete a resource.

- ManagedClusterAction is a namespace-scoped CRD, and the definition is [here](../../deploy/foundation/hub/resources/crds/action.open-cluster-management.io_managedclusteractions.crd.yaml).
- ManagedClusterAction should be applied in the namespace of a certain managed cluster, and the usage examples is [here](../../examples/action).

## CRD Spec

```yaml
apiVersion: action.open-cluster-management.io/v1beta1
kind: ManagedClusterAction
metadata:
  name: <CR name>
  namespace: <namespace of managed cluster>
spec:
  actionType: <Create/Update/Delete>
  kube:
    <kube resource payload>
status:
  conditions:
  - lastTransitionTime: "2020-05-09T10:05:17Z"
    status: "False"
    type: Completed
    reason: ...
    message: ...
  result:
    <references the related result of the action>
```

In the `spec` section:

- `actionType` is the type of action which can be `Create`,`Update` or `Delete`.
- `kube` is the payload of resource to process.

In `status` section:

- `conditions` includes only one condition type `Completed`. The status is `True` if execute the action successfully. Otherwise, the status is `False` if action fails, and the reason is the failure details.
- `result` show the related result of the action.


