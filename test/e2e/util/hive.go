package util

import (
	"context"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hive/v1/aws"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateClusterPool(hiveClient hiveclient.Interface, name, namespace string, labels map[string]string) error {
	clusterPool := &hivev1.ClusterPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
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

	_, err := hiveClient.HiveV1().ClusterPools(namespace).Create(context.TODO(), clusterPool, metav1.CreateOptions{})
	return err
}

func CreateClusterClaim(hiveClient hiveclient.Interface, name, namespace, clusterPool string) error {
	clusterClaim := &hivev1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: hivev1.ClusterClaimSpec{
			ClusterPoolName: clusterPool,
		},
	}
	_, err := hiveClient.HiveV1().ClusterClaims(namespace).Create(context.TODO(), clusterClaim, metav1.CreateOptions{})
	return err
}

func CreateClusterDeployment(hiveClient hiveclient.Interface, name, namespace, clusterPoolName, clusterPoolNamespace string) error {
	clusterDeployment := &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: hivev1.ClusterDeploymentSpec{
			BaseDomain: "dev04.red-chesterfield.com",
			ClusterPoolRef: &hivev1.ClusterPoolReference{
				Namespace: clusterPoolNamespace,
				PoolName:  clusterPoolName,
			},
			Platform: hivev1.Platform{
				AWS: &aws.Platform{
					CredentialsSecretRef: v1.LocalObjectReference{
						Name: "aws-clusterpool-aws-creds",
					},
					Region: "us-east",
				},
			},
			Provisioning: &hivev1.Provisioning{
				InstallConfigSecretRef: &v1.LocalObjectReference{
					Name: "secret-ref",
				},
			},
			PullSecretRef: &v1.LocalObjectReference{
				Name: "pull-ref",
			},
		},
	}
	_, err := hiveClient.HiveV1().ClusterDeployments(namespace).Create(context.TODO(), clusterDeployment, metav1.CreateOptions{})
	return err
}
