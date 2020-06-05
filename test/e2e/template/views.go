package template

// ManagedClusterViewTemplate is json template for namespace
const ManagedClusterViewTemplate = `{
  "apiVersion": "view.open-cluster-management.io/v1beta1",
  "kind": "ManagedClusterView",
  "metadata": {
    "labels": {
      "name": "getdeployment"
    },
    "name": "getdeployment",
    "namespace": "cluster0"
  },
  "spec": {
    "scope": {
      "resource": "deployments",
      "name": "acm-proxyserver",
      "namespace": "open-cluster-management"
    }
  }
}`
