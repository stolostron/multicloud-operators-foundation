package util

import (
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hive/v1/aws"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var NamespaceRandomTemplate = &v1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		GenerateName: "test-automation-",
		Labels: map[string]string{
			"test-automation": "true",
		},
	},
}

var ManagedClusterSetRandomTemplate = &clusterv1alpha1.ManagedClusterSet{
	ObjectMeta: metav1.ObjectMeta{
		GenerateName: "test-automation-clusterset-",
	},
	Spec: clusterv1alpha1.ManagedClusterSetSpec{},
}
var ManagedClusterSetTemplate = &clusterv1alpha1.ManagedClusterSet{
	ObjectMeta: metav1.ObjectMeta{
		Name: "clusterset1",
	},
	Spec: clusterv1alpha1.ManagedClusterSetSpec{},
}

var ClusterRoleBindingAdminTemplate = &rbacv1.ClusterRoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: "clustersetrolebindingAdmin",
	},
	RoleRef: rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "open-cluster-management:managedclusterset:admin:clusterset1",
	},
	Subjects: []rbacv1.Subject{
		{
			Kind:     "User",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "admin1",
		},
	},
}

var ClusterRoleBindingViewTemplate = &rbacv1.ClusterRoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: "clustersetrolebindingView",
	},
	RoleRef: rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "open-cluster-management:managedclusterset:view:clusterset1",
	},
	Subjects: []rbacv1.Subject{
		{
			Kind:     "User",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "view1",
		},
	},
}

var ClusterpoolTemplate = &hivev1.ClusterPool{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "clusterpool1",
		Namespace: "default",
		Labels: map[string]string{
			"cluster.open-cluster-management.io/clusterset": "clusterset1",
		},
	},
	Spec: hivev1.ClusterPoolSpec{
		BaseDomain: "dev04.red-chesterfield.com",
		ImageSetRef: hivev1.ClusterImageSetReference{
			Name: "img4.6.29-x86-64-appsub",
		},
		Platform: hivev1.Platform{
			AWS: &aws.Platform{
				CredentialsSecretRef: v1.LocalObjectReference{
					Name: "aws-clusterpool-aws-creds",
				},
				Region: "us-east",
			},
		},
		Size: 2,
	},
}

var ClusterclaimTemplate = &hivev1.ClusterClaim{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "clusterclaim1",
		Namespace: "default",
	},
	Spec: hivev1.ClusterClaimSpec{
		ClusterPoolName: "clusterpool1",
	},
}
var ClusterdeploymentTemplate = &hivev1.ClusterDeployment{
	ObjectMeta: metav1.ObjectMeta{
		Name: "clusterdeployment1",
	},
	Spec: hivev1.ClusterDeploymentSpec{
		BaseDomain: "dev04.red-chesterfield.com",
		ClusterPoolRef: &hivev1.ClusterPoolReference{
			Namespace: "default",
			PoolName:  "clusterpool1",
		},
		Platform: hivev1.Platform{
			AWS: &aws.Platform{
				CredentialsSecretRef: v1.LocalObjectReference{
					Name: "aws-clusterpool-aws-creds",
				},
				Region: "us-east",
			},
		},
	},
}

const ManagedClusterSetRandomTemplateJson = `{
  "apiVersion": "cluster.open-cluster-management.io/v1alpha1",
  "kind": "ManagedClusterSet",
  "metadata": {
    "generateName": "test-automation-clusterset-"
  }
}`

// ManagedClusterActionCreateTemplate is json template for create action
const ManagedClusterActionCreateTemplate = `{
  "apiVersion": "action.open-cluster-management.io/v1beta1",
  "kind": "ManagedClusterAction",
  "metadata": {
    "labels": {
	  "test-automation": "true"
    },
	"generateName": "test-automation-action-create-",
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
    "labels": {
      "test-automation": "true"
    },
	"generateName": "test-automation-action-delete-",
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
 	"labels": {
	  "test-automation": "true"
    },
	"generateName": "test-automation-action-update-",
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
      "test-automation": "true"
    },
	"generateName": "test-automation-view-",
    "namespace": "cluster1"
  },
  "spec": {
    "scope": {
      "resource": "deployments",
      "name": "foundation-agent",
      "namespace": "open-cluster-management-agent"
    }
  }
}`

const BMATemplate = `
{
  "apiVersion": "inventory.open-cluster-management.io/v1alpha1",
  "kind": "BareMetalAsset",
  "metadata": {
     "name": "mycluster"
  },
  "spec": {
     "bmc": {
        "address": "localhost",
        "credentialsName": "my-secret"
     },
     "hardwareProfile": "test",
     "Role": "worker",
     "clusterDeployment": {
        "name": "mycluster",
        "namespace": "default"
     }
  }
}`
