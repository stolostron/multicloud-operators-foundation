package template

const ClusterActionCreateTemplate = `{
  "apiVersion": "action.open-cluster-management.io/v1beta1",
  "kind": "ClusterAction",
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

const ClusterActionDeleteTemplate = `{
  "apiVersion": "action.open-cluster-management.io/v1beta1",
  "kind": "ClusterAction",
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

const ClusterActionUpdateTemplate = `{
  "apiVersion": "action.open-cluster-management.io/v1beta1",
  "kind": "ClusterAction",
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
