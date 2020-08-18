
# Overview

ACM Foundation supports some foundational components based ManagedCluster for ACM.

## Hub cluster

* proxyserver: Provide a serverless aggregated API `clusterstatus` to proxy requests from managed clusters to other backend servers.

* controller: Controllers of ManagedClusterView, ManagedClusterAction and ManagedClusterInfo CRDs on Hub Cluster.

* webhook: Currently, webhook is used to add user info to custom resource, which is used in RBAC.

## Managed cluster

* agent: The controllers of ManagedClusterView, ManagedClusterAction and ManagedClusterInfo CRDs on Managed Cluster.
