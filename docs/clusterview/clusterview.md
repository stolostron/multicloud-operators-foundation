# APIs in group clusterview.open-cluster-management.io

There are 2 aggregated APIs in this group for non-admin users to list authorized ManagedClusters and ManagedClusterSets.

The 2 APIs are only used to do `GET`,`LIST` and `WATCH` verbs.

1. managedclusters.clusterview.open-cluster-management.io
2. managedclustersets.clusterview.open-cluster-management.io
 
# Examples

1. Non-admin user Bob can only get ManagedCluster cluster1 and cluster2.

    Create ClusterRole and ClusterRoleBinding for user Bob.
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

    Bob can only list the authorized ManagedClusters
    ```bash
    $ kubectl get managedclusters.clusterview.open-cluster-management.io
    NAME            CREATED AT
    cluster1   2021-03-11T01:32:51Z
    cluster2   2021-03-10T02:12:10Z
    ```


2. Non-admin user Bob can only get ManagedClusterSet clusterset1 and clusterset2.

    Create ClusterRole and ClusterRoleBinding for user Bob.
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
    
    Bob can only list the authorized ManagedClusters
    ```bash
    $ kubectl get managedclustersets.clusterview.open-cluster-management.io
    NAME            CREATED AT
    clusterset1   2021-03-11T05:32:31Z
    clusterset2   2021-03-10T03:33:10Z
    ```
