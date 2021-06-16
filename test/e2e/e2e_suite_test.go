package e2e

import (
	"context"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	"os"
	"testing"

	clustersetutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/clusterset"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"

	addonv1alpha1client "github.com/open-cluster-management/api/client/addon/clientset/versioned"
	clusterclient "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	apiregistrationclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
)

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junit_report_file := os.Getenv("JUNIT_REPORT_FILE")
	if junit_report_file != "" {
		junitReporter := reporters.NewJUnitReporter(junit_report_file)
		ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "E2E suite", []ginkgo.Reporter{junitReporter})
	} else {
		ginkgo.RunSpecs(t, "E2E suite")
	}
}

const (
	eventuallyTimeout  = 300
	eventuallyInterval = 2
)

var (
	dynamicClient          dynamic.Interface
	kubeClient             kubernetes.Interface
	hiveClient             hiveclient.Interface
	clusterClient          clusterclient.Interface
	addonClient            addonv1alpha1client.Interface
	apiRegistrationClient  *apiregistrationclient.ApiregistrationV1Client
	managedClusterName     string
	managedClusterSetName  = "clusterset1"
	fakeManagedClusterName = util.RandomName()
)

// This suite is sensitive to the following environment variables:
//
// - KUBECONFIG is the location of the kubeconfig file to use
// - MANAGED_CLUSTER_NAME is the name of managed cluster that is deployed by registration-operator
var _ = ginkgo.BeforeSuite(func() {
	var err error

	managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
	if managedClusterName == "" {
		managedClusterName = "cluster1"
	}

	dynamicClient, err = util.NewDynamicClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	kubeClient, err = util.NewKubeClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	hiveClient, err = util.NewHiveClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	apiRegistrationClient, err = util.NewAPIServiceClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	cfg, err := util.NewKubeConfig()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	addonClient, err = addonv1alpha1client.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	clusterClient, err = clusterclient.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	gomega.Eventually(func() error {
		return util.CheckFoundationPodsReady()
	}, eventuallyTimeout, 2*eventuallyInterval).Should(gomega.Succeed())

	// accept the managed cluster that is deployed by registration-operator
	err = util.CheckJoinedManagedCluster(clusterClient, managedClusterName)
	if err != nil {
		err = util.AcceptManagedCluster(kubeClient, clusterClient, managedClusterName)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	// import a fake managed cluster
	err = util.ImportManagedCluster(clusterClient, fakeManagedClusterName)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	err = util.CreateManagedClusterSet(clusterClient, managedClusterSetName)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// set managedClusterSet for managedCluster
	clusterSetLabel := map[string]string{
		clustersetutils.ClusterSetLabel: managedClusterSetName,
	}

	err = util.UpdateManagedClusterLabels(clusterClient, managedClusterName, clusterSetLabel)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// create managedClusterSet admin clusterRoleBinding
	_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(context.Background(), util.ClusterRoleBindingAdminTemplate, metav1.CreateOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	// create  managedClusterSet view clusterRoleBinding
	_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(context.Background(), util.ClusterRoleBindingViewTemplate, metav1.CreateOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
})

var _ = ginkgo.AfterSuite(func() {
	// delete fake cluster
	err := util.CleanManagedCluster(clusterClient, fakeManagedClusterName)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	// delete managedClusterSet
	err = util.DeleteManagedClusterSet(clusterClient, managedClusterSetName)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	// delete managedClusterSet admin/view clusterRoleBinding
	err = kubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), util.ClusterRoleBindingAdminTemplate.Name, metav1.DeleteOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	err = kubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), util.ClusterRoleBindingViewTemplate.Name, metav1.DeleteOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
})
