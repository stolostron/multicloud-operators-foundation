package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	apiregistrationclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"

	"github.com/stolostron/cluster-lifecycle-api/helpers/imageregistry"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"

	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
)

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2E suite")
}

const (
	eventuallyTimeout  = 300
	eventuallyInterval = 2
)

var (
	dynamicClient         dynamic.Interface
	discoveryClient       discovery.DiscoveryInterface
	kubeClient            kubernetes.Interface
	hiveClient            hiveclient.Interface
	clusterClient         clusterclient.Interface
	ocpClient             openshiftclientset.Interface
	addonClient           addonv1alpha1client.Interface
	apiRegistrationClient *apiregistrationclient.ApiregistrationV1Client
	imageRegistryClient   imageregistry.Interface
	defaultManagedCluster string
	foundationNS          string
	isOcp                 = false
	hubVersionInfo        *version.Info
)

// This suite is sensitive to the following environment variables:
//
// - KUBECONFIG is the location of the kubeconfig file to use
// - MANAGED_CLUSTER_NAME is the name of managed cluster that is deployed by registration-operator
var _ = ginkgo.BeforeSuite(func() {
	var err error

	defaultManagedCluster = os.Getenv("MANAGED_CLUSTER_NAME")
	if defaultManagedCluster == "" {
		defaultManagedCluster = "cluster1"
	}

	foundationNS = os.Getenv("DEPLOY_NAMESPACE")
	if foundationNS == "" {
		foundationNS = "open-cluster-management"
	}

	dynamicClient, err = util.NewDynamicClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	discoveryClient, err = util.NewDiscoveryClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	kubeClient, err = util.NewKubeClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	ocpClient, err = util.NewOCPClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	hiveClient, err = util.NewHiveClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	apiRegistrationClient, err = util.NewAPIServiceClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	imageRegistryClient, err = util.NewImageRegistryClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	cfg, err := util.NewKubeConfig()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	addonClient, err = addonv1alpha1client.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	clusterClient, err = clusterclient.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// accept the default managed cluster
	err = util.CheckJoinedManagedCluster(clusterClient, defaultManagedCluster)
	if err != nil {
		err = util.AcceptManagedCluster(kubeClient, clusterClient, defaultManagedCluster)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	hubVersionInfo, err = discoveryClient.ServerVersion()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	if hubVersionInfo != nil {
		ginkgo.By("hub version: " + hubVersionInfo.String())
	}

	gomega.Eventually(func() error {
		_, err = kubeClient.AppsV1().Deployments("open-cluster-management-agent-addon").Get(context.TODO(), "klusterlet-addon-workmgr", metav1.GetOptions{})
		return err
	}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() error {
		managedClusterInfo, err := util.GetResource(dynamicClient, util.ClusterInfoGVR, defaultManagedCluster, defaultManagedCluster)
		if err != nil {
			return err
		}
		// check the distributionInfo
		isOcp, err = util.IsOCP(managedClusterInfo)
		if err != nil {
			return err
		}
		return nil
	}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
})
