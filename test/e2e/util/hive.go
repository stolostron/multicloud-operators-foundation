package util

import (
	"context"
	"fmt"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hive/v1/aws"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	hivescheme = runtime.NewScheme()
)

func init() {
	_ = hivev1.AddToScheme(hivescheme)
}

func NewHiveClient() (client.Client, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{Scheme: hivescheme})
}

func NewHiveClientWithImpersonate(user string, groups []string) (client.Client, error) {
	cfg, err := NewKubeConfig()
	if err != nil {
		return nil, err
	}

	cfg.Impersonate.UserName = user
	cfg.Impersonate.Groups = groups

	return client.New(cfg, client.Options{Scheme: hivescheme})
}

func CreateClusterPool(hiveClient client.Client, name, namespace string, labels map[string]string) error {
	err := CreateNamespace(namespace)
	if err != nil {
		return err
	}
	clusterPool := &hivev1.ClusterPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    labels,
			Namespace: namespace,
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

	return hiveClient.Create(context.TODO(), clusterPool)
}

func UpdateClusterPoolLabel(hiveClient client.Client, name, namespace string, labels map[string]string) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		oldPool := &hivev1.ClusterPool{}
		err := hiveClient.Get(context.TODO(), types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, oldPool)
		if err != nil {
			return err
		}
		if oldPool.Labels != nil {
			for k, v := range labels {
				oldPool.Labels[k] = v
			}
		} else {
			oldPool.Labels = labels
		}
		return hiveClient.Update(context.TODO(), oldPool)
	})
}

func CreateClusterClaim(hiveClient client.Client, name, namespace, clusterPool string) error {
	clusterClaim := &hivev1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: hivev1.ClusterClaimSpec{
			ClusterPoolName: clusterPool,
		},
	}
	return hiveClient.Create(context.TODO(), clusterClaim)
}

func CreateClusterDeployment(hiveClient client.Client, name, namespace, clusterPoolName, clusterPoolNamespace string, labels map[string]string, fromAi bool) error {
	err := CreateNamespace(namespace)
	if err != nil {
		return err
	}
	var poolRef = &hivev1.ClusterPoolReference{}
	if clusterPoolName != "" {
		poolRef = &hivev1.ClusterPoolReference{
			Namespace: clusterPoolNamespace,
			PoolName:  clusterPoolName,
		}
	} else {
		poolRef = nil
	}
	clusterDeployment := &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: hivev1.ClusterDeploymentSpec{
			BaseDomain:     "dev04.red-chesterfield.com",
			ClusterPoolRef: poolRef,
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
	if fromAi {
		clusterDeployment.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				Kind:       "AgentCluster",
				APIVersion: "capi-provider.agent-install.openshift.io/v1alpha1",
				Name:       "fake-agent-cluster",
				UID:        "fake-uid-12345",
			},
		}
	}
	return hiveClient.Create(context.TODO(), clusterDeployment)
}

func UpdateClusterDeploymentLabels(hiveClient client.Client, name, namespace string, labels map[string]string) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		oldClusterDeployment := &hivev1.ClusterDeployment{}
		err := hiveClient.Get(context.TODO(), types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, oldClusterDeployment)
		if err != nil {
			return err
		}
		if oldClusterDeployment.Labels != nil {
			for k, v := range labels {
				oldClusterDeployment.Labels[k] = v
			}
		} else {
			oldClusterDeployment.Labels = labels
		}

		err = hiveClient.Update(context.TODO(), oldClusterDeployment)
		return err
	})
}

func CleanClusterDeployment(hiveClient client.Client, name, namespace string) error {
	err := hiveClient.Delete(context.TODO(), &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return DeleteNamespace(namespace)
}

func CleanClusterPool(hiveClient client.Client, name, namespace string) error {
	err := hiveClient.Delete(context.TODO(), &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return DeleteNamespace(namespace)
}

func ClaimCluster(hiveClient client.Client, cdName, cdNs, claimName string) error {
	cd := &hivev1.ClusterDeployment{}
	err := hiveClient.Get(context.TODO(), types.NamespacedName{
		Name:      cdName,
		Namespace: cdNs,
	}, cd)
	if err != nil {
		return err
	}

	if cd.Spec.ClusterPoolRef == nil {
		return fmt.Errorf("ClusterPoolRef is nil")
	}

	cd.Spec.ClusterPoolRef.ClaimName = claimName

	return hiveClient.Update(context.Background(), cd)
}
