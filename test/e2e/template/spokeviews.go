package template

// SpokeViewTemplate is json template for namespace
const SpokeViewTemplate = `{
  "apiVersion": "view.open-cluster-management.io/v1beta1",
  "kind": "SpokeView",
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
      "namespace": "multicloud-system"
    }
  }
}`
