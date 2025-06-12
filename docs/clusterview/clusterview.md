# APIs in group clusterview.open-cluster-management.io

There are 3 aggregated APIs in this group for users to list authorized ManagedClusters, ManagedClusterSets and KubeVirt Projects.

## `managedclusters.clusterview.open-cluster-management.io` API

The `managedclusters.clusterview.open-cluster-management.io` API is only used to do `GET`,`LIST` and `WATCH` verbs for `ManagedClusters` by non-admin users.

### Example

Non-admin user Bob can only get ManagedCluster cluster1 and cluster2.

1. Create ClusterRole and ClusterRoleBinding for user Bob.
    ```yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: clusterRoleForBob
    rules:
    - apiGroups: ["clusterview.open-cluster-management.io"]
      resources: ["managedclusters"]
      verbs: ["list","get","watch"]
    - apiGroups: ["cluster.open-cluster-management.io"]
      resources: ["managedclusters"]
      verbs: ["get"]
      resourceNames: ["cluster1","cluster2"]
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: clusterRoleBindingForBob
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: clusterRoleForBob
    subjects:
      - kind: User
        apiGroup: rbac.authorization.k8s.io
        name: Bob
    ```

2. Bob can only list the authorized ManagedClusters
    ```bash
    $ kubectl get managedclusters.clusterview.open-cluster-management.io
    NAME            CREATED AT
    cluster1   2021-03-11T01:32:51Z
    cluster2   2021-03-10T02:12:10Z
    ```

## `managedclustersets.clusterview.open-cluster-management.io` API

The `managedclustersets.clusterview.open-cluster-management.io` API is only used to do `GET`,`LIST` and `WATCH` verbs for `ManagedClusterSets` by non-admin users.

### Example

Non-admin user Bob can only get ManagedClusterSet clusterset1 and clusterset2.

1. Create ClusterRole and ClusterRoleBinding for user Bob.
    ```yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: clusterRoleForBob
    rules:
    - apiGroups: ["clusterview.open-cluster-management.io"]
      resources: ["managedclustersets"]
      verbs: ["list","get","watch"]
    - apiGroups: ["cluster.open-cluster-management.io"]
      resources: ["managedclustersets"]
      verbs: ["get"]
      resourceNames: ["clusterset1","clusterset2"]
    
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: clusterRoleBindingForBob
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: clusterRoleForBob
    subjects:
      - kind: User
        apiGroup: rbac.authorization.k8s.io
        name: Bob
    ```

2. Bob can only list the authorized ManagedClusters
    ```bash
    $ kubectl get managedclustersets.clusterview.open-cluster-management.io
    NAME            CREATED AT
    clusterset1   2021-03-11T05:32:31Z
    clusterset2   2021-03-10T03:33:10Z
    ```

## `kubevirtprojects.clusterview.open-cluster-management.io` API

The `kubevirtprojects.clusterview.open-cluster-management.io` API is only used to do `LIST` verbs for KubeVirt projects based on `ClusterPermission` API

### Examples

##### List a user's KubeVirt projects based on this user's clusterpermission

1. The ACM admin creates ClusterRole and ClusterRoleBinding for user Bob.
    ```yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: clusterRoleForBob
    rules:
    - apiGroups: ["clusterview.open-cluster-management.io"]
      resources: ["kubevirtprojects"]
      verbs: ["list"]
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: clusterRoleBindingForBob
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: clusterRoleForBob
    subjects:
      - kind: User
        apiGroup: rbac.authorization.k8s.io
        name: Bob
    ```

2. The ACM admin creates `ClusterPermission` in Bob's cluster namespaces
    ```yaml
    apiVersion: rbac.open-cluster-management.io/v1alpha1
    kind: ClusterPermission
    metadata:
      name: kubevirt-edit
      namespace: cluster1
    spec:
      roleBindings:
      - namespace: kubevirt-workspace-1
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: kubevirt.io:edit
        subject:
          kind: User
          name: Bob
          apiGroup: rbac.authorization.k8s.io
      - namespace: kubevirt-workspace-2
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: kubevirt.io:edit
        subject:
          kind: User
          name: Bob
          apiGroup: rbac.authorization.k8s.io
    ---
    apiVersion: rbac.open-cluster-management.io/v1alpha1
    kind: ClusterPermission
    metadata:
      name: kubevirt-edit
      namespace: cluster2
    spec:
      roleBindings:
      - namespace: kubevirt-workspace-1
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: kubevirt.io:edit
        subject:
          kind: User
          name: Bob
          apiGroup: rbac.authorization.k8s.io
      - namespace: kubevirt-workspace-2
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: kubevirt.io:edit
        subject:
          kind: User
          name: Bob
          apiGroup: rbac.authorization.k8s.io
    ```

3. Bob list his KubeVirt projects
    ```sh
    kubectl get kubevirtprojects.clusterview.open-cluster-management.io
    ```

    ```
    CLUSTER    PROJECT
    cluster1   kubevirt-workspace-1
    cluster1   kubevirt-workspace-2
    cluster2   kubevirt-workspace-1
    cluster2   kubevirt-workspace-2
    ```

    ```sh
    kubectl get kubevirtprojects.clusterview.open-cluster-management.io -oyaml
    ```

    ```yaml
    apiVersion: v1
    kind: List
    items:
    - apiVersion: clusterview.open-cluster-management.io/v1
      kind: Project
      metadata:
        labels:
          cluster: cluster1
          project: kubevirt-workspace-1
        name: 4120e9fb-7eef-50ca-a61c-c3e7ee9dee78
    - apiVersion: clusterview.open-cluster-management.io/v1
      kind: Project
      metadata:
        labels:
          cluster: cluster1
          project: kubevirt-workspace-2
        name: ed37f473-8bca-5a52-87c3-b0cdb1e1c885
    - apiVersion: clusterview.open-cluster-management.io/v1
      kind: Project
      metadata:
        labels:
          cluster: cluster2
          project: kubevirt-workspace-1
        name: 92c27029-9822-5449-a108-dcd783953fae
    - apiVersion: clusterview.open-cluster-management.io/v1
      kind: Project
      metadata:
        labels:
          cluster: cluster2
          project: kubevirt-workspace-2
        name: db2b7ae7-f5e7-5fd3-8f7f-a29cce5236ab
    ```

##### List a group user's KubeVirt projects based on group's clusterpermission

1. ACM admin creates ClusterRole and ClusterRoleBinding for group `kubevirt-projects-view`
    ```yaml
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: clusterRoleForGroup
    rules:
    - apiGroups: ["clusterview.open-cluster-management.io"]
      resources: ["kubevirtprojects"]
      verbs: ["list"]
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: clusterRoleBindingForBob
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: clusterRoleForGroup
    subjects:
      - kind: Group
        apiGroup: rbac.authorization.k8s.io
        name: kubevirt-projects-view
    ```

2. ACM admin creates `ClusterPermission` in group user's cluster namespaces
    ```yaml
    apiVersion: rbac.open-cluster-management.io/v1alpha1
    kind: ClusterPermission
    metadata:
      name: kubevirt-view
      namespace: cluster1
    spec:
      roleBindings:
      - namespace: kubevirt-workspace-1
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: kubevirt.io:view
        subject:
          apiGroup: rbac.authorization.k8s.io
          kind: Group
          name: kubevirt-projects-view
          apiGroup: rbac.authorization.k8s.io
      - namespace: kubevirt-workspace-2
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: kubevirt.io:view
        subject:
          apiGroup: rbac.authorization.k8s.io
          kind: Group
          name: kubevirt-projects-view
          apiGroup: rbac.authorization.k8s.io
    ---
    apiVersion: rbac.open-cluster-management.io/v1alpha1
    kind: ClusterPermission
    metadata:
      name: kubevirt-view
      namespace: cluster1
    spec:
      roleBindings:
      - namespace: kubevirt-workspace-1
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: kubevirt.io:view
        subject:
          apiGroup: rbac.authorization.k8s.io
          kind: Group
          name: kubevirt-projects-view
          apiGroup: rbac.authorization.k8s.io
      - namespace: kubevirt-workspace-2
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: kubevirt.io:view
        subject:
          apiGroup: rbac.authorization.k8s.io
          kind: Group
          name: kubevirt-projects-view
          apiGroup: rbac.authorization.k8s.io
    ```

3. The group user list the KubeVirt projects
    ```sh
    kubectl get kubevirtprojects.clusterview.open-cluster-management.io
    ```

    ```
    CLUSTER    PROJECT
    cluster1   kubevirt-workspace-1
    cluster1   kubevirt-workspace-2
    cluster2   kubevirt-workspace-1
    cluster2   kubevirt-workspace-2
    ```

    ```sh
    kubectl get kubevirtprojects.clusterview.open-cluster-management.io -oyaml
    ```

    ```yaml
    apiVersion: v1
    kind: List
    items:
    - apiVersion: clusterview.open-cluster-management.io/v1
      kind: Project
      metadata:
        labels:
          cluster: cluster1
          project: kubevirt-workspace-1
        name: 4120e9fb-7eef-50ca-a61c-c3e7ee9dee78
    - apiVersion: clusterview.open-cluster-management.io/v1
      kind: Project
      metadata:
        labels:
          cluster: cluster1
          project: kubevirt-workspace-2
        name: ed37f473-8bca-5a52-87c3-b0cdb1e1c885
    - apiVersion: clusterview.open-cluster-management.io/v1
      kind: Project
      metadata:
        labels:
          cluster: cluster2
          project: kubevirt-workspace-1
        name: 92c27029-9822-5449-a108-dcd783953fae
    - apiVersion: clusterview.open-cluster-management.io/v1
      kind: Project
      metadata:
        labels:
          cluster: cluster2
          project: kubevirt-workspace-2
        name: db2b7ae7-f5e7-5fd3-8f7f-a29cce5236ab
    ```

**Note**: If using ClusterRoleBinding in ClusterPermission to bind a kubevirt ClusterRole, this API will return `*` in project field and the project label will be `all_projects`, for example:

```yaml
apiVersion: rbac.open-cluster-management.io/v1alpha1
kind: ClusterPermission
metadata:
  name: kubevirt-admin
  namespace: cluster1
spec:
  clusterRoleBinding:
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: kubevirt.io:admin
    subject:
      kind: Group
      name: system:cluster-admins
      apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.open-cluster-management.io/v1alpha1
kind: ClusterPermission
metadata:
  name: kubevirt-admin
  namespace: cluster2
spec:
  clusterRoleBinding:
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: kubevirt.io:admin
    subject:
      kind: Group
      name: system:cluster-admins
      apiGroup: rbac.authorization.k8s.io
```

```sh
kubectl get kubevirtprojects.clusterview.open-cluster-management.io
```

```
CLUSTER    PROJECT
cluster1   *
cluster2   *
```

```sh
kubectl get kubevirtprojects.clusterview.open-cluster-management.io -oyaml
```

```yaml
apiVersion: v1
kind: List
items:
- apiVersion: clusterview.open-cluster-management.io/v1
  kind: Project
  metadata:
    labels:
      cluster: cluster1
      project: all_projects
    name: 7a42b467-3107-5e88-bd05-ab0e3fc82efa
- apiVersion: clusterview.open-cluster-management.io/v1
  kind: Project
  metadata:
    labels:
      cluster: cluster2
      project: all_projects
    name: 8ae384ef-db25-5677-8b8e-f43152c1b960
```
