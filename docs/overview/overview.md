
# Overview

ACM Foundation supports some foundational components based ManagedCluster for ACM.

## Hub cluster

* acm-proxyserver: Provide a serverless aggregated API `clusterstatus` to proxy requests from managed clusters to other backend servers.

* acm-controller: Controllers of ManagedClusterView, ManagedClusterAction and ManagedClusterInfo CRDs on Hub Cluster.

* acm-webhook: Currently, acm-webhook is used to add user info to custom resource, which is used in RBAC.

## Managed cluster

* acm-agent: The controllers of ManagedClusterView, ManagedClusterAction and ManagedClusterInfo CRDs on Managed Cluster.
