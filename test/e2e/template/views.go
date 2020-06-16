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
    "namespace": "cluster1"
  },
  "spec": {
    "scope": {
      "resource": "deployments",
      "name": "acm-agent",
      "namespace": "open-cluster-management-agent"
    }
  }
}`
