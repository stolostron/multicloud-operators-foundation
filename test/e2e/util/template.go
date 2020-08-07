package util

// ManagedClusterActionCreateTemplate is json template for create action
const ManagedClusterActionCreateTemplate = `{
  "apiVersion": "action.open-cluster-management.io/v1beta1",
  "kind": "ManagedClusterAction",
  "metadata": {
    "name": "nginx-action-create",
    "namespace": "cluster1"
  },
  "spec": {
    "actionType": "Create",
    "kube": {
      "resource": "deployment",
      "namespace": "default",
      "template": {
        "apiVersion": "apps/v1",
        "kind": "Deployment",
        "metadata": {
          "name": "nginx-deployment-action"
        },
        "spec": {
          "selector": {
            "matchLabels": {
              "app": "nginx"
            }
          },
          "replicas": 2,
          "template": {
            "metadata": {
              "labels": {
                "app": "nginx"
              }
            },
            "spec": {
              "containers": [
                {
                  "name": "nginx",
                  "image": "nginx:1.7.9",
                  "ports": [
                    {
                      "containerPort": 80
                    }
                  ]
                }
              ]
            }
          }
        }
      }
    }
  }
}`

// ManagedClusterActionDeleteTemplate is json template for delete action
const ManagedClusterActionDeleteTemplate = `{
  "apiVersion": "action.open-cluster-management.io/v1beta1",
  "kind": "ManagedClusterAction",
  "metadata": {
    "name": "nginx-action-delete",
    "namespace": "cluster1"
  },
  "spec": {
    "actionType": "Delete",
    "kube": {
      "resource": "deployment",
      "namespace": "default",
      "name": "nginx-deployment-action"
    }
  }
}`

// ManagedClusterActionUpdateTemplate is json template for update action
const ManagedClusterActionUpdateTemplate = `{
  "apiVersion": "action.open-cluster-management.io/v1beta1",
  "kind": "ManagedClusterAction",
  "metadata": {
    "name": "nginx-action-update",
    "namespace": "cluster1"
  },
  "spec": {
    "actionType": "Update",
    "kube": {
      "resource": "deployment",
      "namespace": "default",
      "template": {
        "apiVersion": "apps/v1",
        "kind": "Deployment",
        "metadata": {
          "name": "nginx-deployment-action"
        },
        "spec": {
          "selector": {
            "matchLabels": {
              "app": "nginx"
            }
          },
          "replicas": 1,
          "template": {
            "metadata": {
              "labels": {
                "app": "nginx"
              }
            },
            "spec": {
              "containers": [
                {
                  "name": "nginx",
                  "image": "nginx:1.7.9",
                  "ports": [
                    {
                      "containerPort": 80
                    }
                  ]
                }
              ]
            }
          }
        }
      }
    }
  }
}`

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

// NamespaceTemplate is json template for namespace
const NamespaceTemplate = `{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
      "labels": {
    "test-automation": "true"
  },
      "generateName": "test-automation-"
  }
}`

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
