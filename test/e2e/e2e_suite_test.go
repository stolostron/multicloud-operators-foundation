package e2e

import (
	"os"
	"testing"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"

	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	apiregistrationclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
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
	dynamicClient          dynamic.Interface
	kubeClient             kubernetes.Interface
	hiveClient             hiveclient.Interface
	apiRegistrationClient  *apiregistrationclient.ApiregistrationV1Client
	managedClusterName     string
	fakeManagedClusterName string
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

	// accept the managed cluster that is deployed by registration-operator
	err = util.AcceptManagedCluster(managedClusterName)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// create a fake managed cluster
	fakeManagedCluster, err := util.CreateManagedCluster(dynamicClient)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	fakeManagedClusterName = fakeManagedCluster.GetName()

	gomega.Eventually(func() error {
		return util.CheckFoundationPodsReady()
	}, 60*time.Second, 2*time.Second).Should(gomega.Succeed())

})
