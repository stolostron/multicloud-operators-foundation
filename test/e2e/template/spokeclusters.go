package template

// SpokeClusterTemplate is json template for namespace
const SpokeClusterTemplate = `{
  "apiVersion": "cluster.open-cluster-management.io/v1",
  "kind": "SpokeCluster",
  "metadata": {
    "name": "cluster1"
  },
  "spec": {
    "hubAcceptsClient": true,
    "spokeClientConfigs": [
      {
        "caBundle": "test",
        "url": "https://test.com"
      }
    ]
  }
}`
