# APIs in group clusterview.open-cluster-management.io

There are 4 aggregated APIs in this group for users to list authorized ManagedClusters, ManagedClusterSets, KubeVirt Projects, and User Permissions.

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

**Note**: If using ClusterRoleBinding in ClusterPermission to bind a kubevirt ClusterRole, this API will return `*` in project field and the project label will be `all_projects`.

ClusterPermission supports both `clusterRoleBinding` (singular, single binding) and `clusterRoleBindings` (plural, array of bindings) fields. You can use either field based on your needs.

##### Example using `clusterRoleBinding` (singular) field

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

#### Example using `clusterRoleBindings` (plural) field

```yaml
apiVersion: rbac.open-cluster-management.io/v1alpha1
kind: ClusterPermission
metadata:
  name: kubevirt-admin
  namespace: cluster1
spec:
  clusterRoleBindings:
  - roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: kubevirt.io:admin
    subject:
      kind: Group
      name: system:cluster-admins
      apiGroup: rbac.authorization.k8s.io
  - roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: kubevirt.io:edit
    subject:
      kind: User
      name: Alice
      apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.open-cluster-management.io/v1alpha1
kind: ClusterPermission
metadata:
  name: kubevirt-admin
  namespace: cluster2
spec:
  clusterRoleBindings:
  - roleRef:
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

## `userpermissions.clusterview.open-cluster-management.io` API

The `userpermissions.clusterview.open-cluster-management.io` API allows users to discover their permissions across the fleet of managed clusters. This API returns both ClusterRole definitions and their cluster/namespace bindings, providing a complete view of what permissions a user has and where they can use them.

### Overview

The UserPermission API is a label-based discovery system where:

- ACM administrators define ClusterRoles with specific labels (`clusterview.open-cluster-management.io/discoverable: "true"`)
- Users can query their permissions by calling the aggregated API
- The system determines permissions by reading ClusterPermission resources on the hub cluster
- Both individual user permissions and group-based permissions are supported

### How It Works

1. **Labeled ClusterRoles**: Administrators create ClusterRoles with the discoverable label
2. **Permission Grant**: ClusterPermission resources bind users/groups to labeled ClusterRoles on specific clusters/namespaces
3. **API Access**: Administrators grant users permission to call the userpermissions API
4. **Discovery**: Users call the API to see which ClusterRoles they have access to and where

### API Operations

The API supports:

- `LIST`: Get all ClusterRoles the user has bindings to
- `GET`: Get a specific ClusterRole by name (returns 404 if user has no bindings to it)

Both operations only return ClusterRoles that the requesting user actually has permissions for.

### Examples

#### Example 1: Basic Setup - Granting Permissions to User Bob

**Step 1**: Administrator creates labeled ClusterRoles

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: acm-kubevirt.io:admin
  labels:
    clusterview.open-cluster-management.io/discoverable: "true"
rules:
- apiGroups: ["kubevirt.io"]
  resources: ["virtualmachines", "virtualmachineinstances"]
  verbs: ["get", "list", "create", "update", "delete"]
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "create", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: acm-deployments-admin
  labels:
    clusterview.open-cluster-management.io/discoverable: "true"
rules:
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "create", "update", "delete"]
- apiGroups: [""]
  resources: ["configmaps", "serviceaccounts"]
  verbs: ["get", "list", "create", "update", "delete"]
```

**Step 2**: Administrator grants Bob permissions via ClusterPermission resources

```yaml
# Grant cluster-wide kubevirt admin on cluster1
apiVersion: rbac.open-cluster-management.io/v1alpha1
kind: ClusterPermission
metadata:
  name: bob-kubevirt-admin-permission
  namespace: cluster1
spec:
  clusterRoleBindings:
    - name: bob-kubevirt-admin
      subjects:
        - kind: User
          apiGroup: rbac.authorization.k8s.io
          name: Bob
      roleRef:
        apiGroup: rbac.authorization.k8s.io
        kind: ClusterRole
        name: acm-kubevirt.io:admin
---
# Grant namespace-scoped deployments admin on cluster1
apiVersion: rbac.open-cluster-management.io/v1alpha1
kind: ClusterPermission
metadata:
  name: bob-deployments-admin-permission
  namespace: cluster1
spec:
  roleBindings:
    - name: bob-deployments-admin
      namespace: app-ns
      subjects:
        - kind: User
          apiGroup: rbac.authorization.k8s.io
          name: Bob
      roleRef:
        apiGroup: rbac.authorization.k8s.io
        kind: ClusterRole
        name: acm-deployments-admin
```

**Step 3**: Administrator grants Bob API access

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: userpermissions-reader
rules:
- apiGroups: ["clusterview.open-cluster-management.io"]
  resources: ["userpermissions"]
  verbs: ["list", "get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: bob-userpermissions-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: userpermissions-reader
subjects:
  - kind: User
    apiGroup: rbac.authorization.k8s.io
    name: Bob
```

**Step 4**: Bob lists his permissions

```bash
kubectl get userpermissions.clusterview.open-cluster-management.io
```

Output:

```
NAME                   BINDINGS
acm-kubevirt.io:admin  cluster1(*)
acm-deployments-admin  cluster1(app-ns)
```

Get detailed information:

```bash
kubectl get userpermissions.clusterview.open-cluster-management.io -oyaml
```

Output:

```yaml
apiVersion: v1
kind: List
items:
- metadata:
    name: acm-kubevirt.io:admin
  status:
    bindings:
    - cluster: cluster1
      scope: cluster
      namespaces: ["*"]
    clusterRoleDefinition:
      rules:
      - apiGroups: ["kubevirt.io"]
        resources: ["virtualmachines", "virtualmachineinstances"]
        verbs: ["get", "list", "create", "update", "delete"]
      - apiGroups: [""]
        resources: ["configmaps", "secrets"]
        verbs: ["get", "list", "create", "update"]
- metadata:
    name: acm-deployments-admin
  status:
    bindings:
    - cluster: cluster1
      scope: namespace
      namespaces: ["app-ns"]
    clusterRoleDefinition:
      rules:
      - apiGroups: ["apps"]
        resources: ["deployments"]
        verbs: ["get", "list", "create", "update", "delete"]
      - apiGroups: [""]
        resources: ["configmaps", "serviceaccounts"]
        verbs: ["get", "list", "create", "update", "delete"]
```

#### Example 2: Group-Based Permissions

**Step 1**: Administrator grants permissions to a group

```yaml
apiVersion: rbac.open-cluster-management.io/v1alpha1
kind: ClusterPermission
metadata:
  name: team-deployments-permission
  namespace: cluster1
spec:
  roleBindings:
    - name: team-deployments-admin
      namespace: app-ns
      subjects:
        - kind: Group
          apiGroup: rbac.authorization.k8s.io
          name: development-team
      roleRef:
        apiGroup: rbac.authorization.k8s.io
        kind: ClusterRole
        name: acm-deployments-admin
```

**Step 2**: Grant API access to the group

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: team-userpermissions-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: userpermissions-reader
subjects:
  - kind: Group
    apiGroup: rbac.authorization.k8s.io
    name: development-team
```

**Step 3**: Any member of the development-team group can now query their permissions and will see the permissions granted to the group.

#### Example 3: ManagedClusterAdmin/ManagedClusterView Integration

The UserPermission API automatically generates synthetic `managedcluster:admin` and `managedcluster:view` permission entries based on RBAC permissions:

- **managedcluster:admin**: Granted when a user has **both** `create` permission on `managedclusteractions` (action.open-cluster-management.io) AND `managedclusterviews` (view.open-cluster-management.io) resources
- **managedcluster:view**: Granted when a user has `create` permission on `managedclusterviews` (view.open-cluster-management.io) resources

**Known Limitation**: The `managedcluster:admin` role requires both permissions to be granted by a **single** ClusterRole or Role. If a user has permissions from multiple separate RoleBindings/ClusterRoleBindings (e.g., one binding granting `managedclusteractions` create and another binding granting `managedclusterviews` create), they will only receive the `managedcluster:view` role, not the `managedcluster:admin` role. To grant admin permissions, create a single ClusterRole or Role that includes both permission rules.

**User with managedclusteradmin on cluster1 and cluster2, plus kubevirt-admin on cluster1:**

```bash
kubectl get userpermissions.clusterview.open-cluster-management.io -oyaml
```

Output:

```yaml
apiVersion: v1
kind: List
items:
- metadata:
    name: managedcluster:admin
  status:
    bindings:
    - cluster: cluster1
      scope: cluster
      namespaces: ["*"]
    - cluster: cluster2
      scope: cluster
      namespaces: ["*"]
    clusterRoleDefinition:
      rules:
      - apiGroups: ["*"]
        resources: ["*"]
        verbs: ["*"]
- metadata:
    name: acm-kubevirt.io:admin
  status:
    bindings:
    - cluster: cluster1
      scope: cluster
      namespaces: ["*"]
    clusterRoleDefinition:
      rules:
      - apiGroups: ["kubevirt.io"]
        resources: ["virtualmachines", "virtualmachineinstances"]
        verbs: ["get", "list", "create", "update", "delete"]
      # ... (additional rules)
```

#### Example 4: Get Specific ClusterRole

To get information about a specific ClusterRole:

```bash
kubectl get userpermissions.clusterview.open-cluster-management.io acm-kubevirt.io:admin -oyaml
```

This returns details for that specific ClusterRole if the user has bindings to it, or returns a 404 Not Found error if the user has no bindings to that ClusterRole.

### Important Notes

1. **Discoverable Label Required**: Only ClusterRoles with the label `clusterview.open-cluster-management.io/discoverable: "true"` are discoverable through this API.

2. **ClusterRole References Only**: The API only considers ClusterPermissions that bind users/groups to ClusterRoles. Role definitions or rule-based permissions in ClusterPermissions are ignored.

3. **Group Membership Assumption**: The system assumes that hub and managed clusters share the same identity providers, so group memberships are consistent across hub and managed clusters.

4. **Performance**: The API uses an efficient caching layer to minimize load on the API server. Permissions are pre-computed rather than queried in real-time.

5. **Security**: Users only see ClusterRoles they have actual permissions for, following standard Kubernetes API conventions.

6. **Response Size**: When users have access to many large ClusterRoles, LIST operations can result in large response payloads. In such cases, use GET operations for specific ClusterRoles to get targeted information.
