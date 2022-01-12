package e2e

import (
	"context"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	clusterinfov1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

var _ = ginkgo.Describe("Testing Pod log", func() {
	podNamespace := "open-cluster-management-agent"
	var podName string
	var containerName string
	var restClient *rest.RESTClient

	ginkgo.BeforeEach(func() {
		// build rest client to get logs
		cfg, err := util.NewKubeConfig()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		gv := v1beta1.SchemeGroupVersion
		cfg.GroupVersion = &gv
		cfg.APIPath = "/apis"
		cfg.NegotiatedSerializer = resource.UnstructuredPlusDefaultContentConfig().NegotiatedSerializer
		restClient, err = rest.RESTClientFor(cfg)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// make sure the api service v1.admission.cluster.open-cluster-management.io is available
		gomega.Eventually(func() bool {
			apiService, err := apiRegistrationClient.APIServices().Get(context.TODO(), "v1beta1.proxy.open-cluster-management.io", metav1.GetOptions{})
			if err != nil {
				return false
			}
			if len(apiService.Status.Conditions) == 0 {
				return false
			}
			return apiService.Status.Conditions[0].Type == apiregistrationv1.Available &&
				apiService.Status.Conditions[0].Status == apiregistrationv1.ConditionTrue
		}, 60*time.Second, 1*time.Second).Should(gomega.BeTrue())

		// find the first pod in open-cluster-management-agent ns
		gomega.Eventually(func() bool {
			pods, err := kubeClient.CoreV1().Pods(podNamespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return false
			}

			if len(pods.Items) == 0 {
				return false
			}

			podName = pods.Items[0].Name
			if len(pods.Items[0].Spec.Containers) == 0 {
				return false
			}
			containerName = pods.Items[0].Spec.Containers[0].Name
			return true
		}, 60*time.Second, 1*time.Second).Should(gomega.BeTrue())
	})

	ginkgo.It("should get log from pod successfully", func() {
		gomega.Eventually(func() bool {
			exists, err := util.HasResource(dynamicClient, clusterInfoGVR, managedClusterName, managedClusterName)
			if err != nil {
				return false
			}
			return exists
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

		gomega.Eventually(func() bool {
			managedClusterInfo, err := util.GetResource(dynamicClient, clusterInfoGVR, managedClusterName, managedClusterName)
			if err != nil {
				return false
			}
			// check the ManagedClusterInfo status
			return util.GetConditionTypeFromStatus(managedClusterInfo, clusterinfov1beta1.ManagedClusterInfoSynced)
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

		req := restClient.Get().Namespace(managedClusterName).
			Name(managedClusterName).
			Resource("clusterstatuses").
			SubResource("log").Suffix(podNamespace, podName, containerName)

		gomega.Eventually(func() bool {
			_, err := req.DoRaw(context.TODO())

			if err != nil {
				return false
			}
			return true
		}, 60*time.Second, 1*time.Second).Should(gomega.BeTrue())
	})
})
