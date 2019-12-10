# MCM Work API

This document is to summarise work API. Work in mcm is defined as a certain job executed on a certain cluster. Currently we use work api for two purpose:

1. Resources view: klusterlet receives work, queries the resource in managed cluster and return result
2. Action work: Create/Update/Delete kube resource or helm release in managed cluster.

## ViewWork

### API Spec

```yaml
apiVersion: mcm.ibm.com/v1alpha1
kind: Work
metadata:
  name: node
  namespace: mycluster-namespace
spec:
  cluster:
    name: mycluster
  type: Resource
  scope:
    resourceType: pod
    labelSelector:
      app: nginx
status:
  type: completed
  reason: ""
  result: {pod list}
```

In the `spec` section:

- `cluster` specifies the cluster that the work should execute on.
- `type` define the work type, it should be `Resource` or `Action`
- `scope` defines the resource filter on managed cluster to get resources.

In `status` section:

- `type` is the work status type, it should be `Completed`, `Failed`, `Processing`.
- `reason` specifies the error message when work failed.
- `results` show the work result data, it is `runtime.RawExtension` type.

## ActionWork

### Create Kube Resource

```yaml
apiVersion: mcm.ibm.com/v1alpha1
kind: Work
metadata:
  name: create-kube-work
  namespace: mycluster-namespace
spec:
  cluster:
    name: mycluster
  type: Action
  actionType: create
  kube:
    resource: deployment
    template:
      apiVersion: extensions/v1beta1
      kind: Deployment
      metadata:
        labels:
          k8s-app: heapster
          kubernetes.io/cluster-service: "true"
        name: heapster
        namespace: kube-system
      ...
```

### Create Helm Resource

```yaml
apiVersion: mcm.ibm.com/v1alpha1
kind: Work
metadata:
  name: create-helm-work
  namespace: mycluster-namespace
spec:
  cluster:
    name: mycluster
  type: Action
  actionType: Create
  helm:
    releaseName: nginx-lego
    chartURL: https://kubernetes-charts.storage.googleapis.com/nginx-lego-0.3.1.tgz
    namespace: kube-system

```

In the `spec` section:

- `cluster` specifies the cluster that the work should execute on.
- `type` define the work type, it should be `Resource` or `Action`
- `actionType` define what action should be apply in target cluster, it only valid when `type` is `Action`. It should be `Create`, `Delete` or `Update`.
- `kube` specifies the kube resource, it cannot be defined with `helm` at the same time. `kube/template` define the kube resource template, it should be `runtime.RawExtension` type.
- `helm` specifies the helm resource, it cannot be defined with `kube` at the same time.
