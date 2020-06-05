package template

// ManagedClusterTemplate is json template for namespace
const ManagedClusterTemplate = `{
  "apiVersion": "cluster.open-cluster-management.io/v1",
  "kind": "ManagedCluster",
  "metadata": {
    "name": "cluster1"
  },
  "spec": {
    "hubAcceptsClient": true,
    "managedClusterClientConfigs": [
      {
        "caBundle": "test",
        "url": "https://test.com"
      }
    ]
  }
}`
