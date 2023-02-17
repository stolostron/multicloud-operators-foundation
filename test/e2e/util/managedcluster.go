package util

import (
	"context"
	"fmt"
	"time"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func NewManagedCluster(name string) *clusterv1.ManagedCluster {
	return &clusterv1.ManagedCluster{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: clusterv1.ManagedClusterSpec{
			ManagedClusterClientConfigs: []clusterv1.ClientConfig{
				{
					CABundle: []byte("ca"),
					URL:      "https://test.com",
				},
			},
			HubAcceptsClient: false,
		},
	}
}

func NewNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func NewClusterClient() (clusterclient.Interface, error) {
	cfg, err := NewKubeConfig()
	if err != nil {
		return nil, err
	}
	return clusterclient.NewForConfig(cfg)
}

func NewClusterClientWithImpersonate(user string, groups []string) (clusterclient.Interface, error) {
	cfg, err := NewKubeConfig()
	if err != nil {
		return nil, err
	}

	cfg.Impersonate.UserName = user
	cfg.Impersonate.Groups = groups

	return clusterclient.NewForConfig(cfg)
}

func CreateNamespace(name string) error {
	kubeClient, err := NewKubeClient()
	if err != nil {
		return err
	}

	_, err = kubeClient.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = kubeClient.CoreV1().Namespaces().Create(context.TODO(), NewNamespace(name), metav1.CreateOptions{})
			return err
		}
	}

	return err
}

func DeleteNamespace(name string) error {
	kubeClient, err := NewKubeClient()
	if err != nil {
		return err
	}

	err = kubeClient.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func CreateManagedCluster(client clusterclient.Interface, cluster *clusterv1.ManagedCluster) error {
	_, err := client.ClusterV1().ManagedClusters().Create(context.TODO(), cluster, metav1.CreateOptions{})
	return err
}

func UpdateManagedClusterLabels(client clusterclient.Interface, clusterName string, labels map[string]string) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		oldCluster, err := client.ClusterV1().ManagedClusters().Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if oldCluster.Labels != nil {
			for k, v := range labels {
				oldCluster.Labels[k] = v
			}
		} else {
			oldCluster.Labels = labels
		}
		_, err = client.ClusterV1().ManagedClusters().Update(context.TODO(), oldCluster, metav1.UpdateOptions{})
		return err
	})
}

func ImportManagedCluster(client clusterclient.Interface, clusterName string) error {
	if err := CreateNamespace(clusterName); err != nil {
		return err
	}

	return CreateManagedCluster(client, NewManagedCluster(clusterName))
}

func CheckJoinedManagedCluster(client clusterclient.Interface, clusterName string) error {
	cluster, err := client.ClusterV1().ManagedClusters().Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, condition := range cluster.Status.Conditions {
		if condition.Type == clusterv1.ManagedClusterConditionJoined &&
			condition.Status == metav1.ConditionTrue {
			return nil
		}
	}
	return fmt.Errorf("the managedcluster %s was not JOINED", clusterName)
}

func CleanManagedCluster(client clusterclient.Interface, clusterName string) error {
	err := client.ClusterV1().ManagedClusters().Delete(context.TODO(), clusterName, metav1.DeleteOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return DeleteNamespace(clusterName)
}

func AcceptManagedCluster(hubClient kubernetes.Interface, clusterClient clusterclient.Interface, clusterName string) error {
	var (
		csrs      *certificatesv1beta1.CertificateSigningRequestList
		csrClient = hubClient.CertificatesV1beta1().CertificateSigningRequests()
	)
	// Waiting for the CSR for ManagedCluster to exist
	if err := wait.Poll(1*time.Second, 120*time.Second, func() (bool, error) {
		var err error
		csrs, err = csrClient.List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("open-cluster-management.io/cluster-name = %v", clusterName),
		})
		if err != nil {
			return false, err
		}

		if len(csrs.Items) >= 1 {
			return true, nil
		}

		return false, nil
	}); err != nil {
		return err
	}

	// Approving all pending CSRs
	for i := range csrs.Items {
		csr := &csrs.Items[i]

		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if isCSRInTerminalState(&csr.Status) {
				return nil
			}

			csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1beta1.CertificateSigningRequestCondition{
				Type:    certificatesv1beta1.CertificateApproved,
				Reason:  "Approved by E2E",
				Message: "Approved as part of Loopback e2e",
			})
			_, err := csrClient.UpdateApproval(context.TODO(), csr, metav1.UpdateOptions{})
			return err
		}); err != nil {
			return err
		}
	}

	// Waiting for ManagedCluster to exist
	if err := wait.Poll(1*time.Second, 120*time.Second, func() (bool, error) {
		_, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return true, nil
	}); err != nil {
		return err
	}

	// Accepting ManagedCluster
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		managedCluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		managedCluster.Spec.HubAcceptsClient = true
		managedCluster.Spec.LeaseDurationSeconds = 5
		_, err = clusterClient.ClusterV1().ManagedClusters().Update(context.TODO(), managedCluster, metav1.UpdateOptions{})
		return err
	}); err != nil {
		return err
	}

	return nil
}

func isCSRInTerminalState(status *certificatesv1beta1.CertificateSigningRequestStatus) bool {
	for _, c := range status.Conditions {
		if c.Type == certificatesv1beta1.CertificateApproved {
			return true
		}
		if c.Type == certificatesv1beta1.CertificateDenied {
			return true
		}
	}
	return false
}
