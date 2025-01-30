package e2e

import (
	"context"
	"fmt"

	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	clusterinfov1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/proxy/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

var _ = ginkgo.Describe("Testing Cluster Proxy Addon get pod log", func() {
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

		// make sure the managed-serviceAccount and cluster-proxy addons are created and available.
		gomega.Eventually(func() error {
			clusterProxyAddon, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(defaultManagedCluster).Get(context.TODO(), "cluster-proxy", metav1.GetOptions{})
			if err != nil {
				return err
			}
			if !meta.IsStatusConditionPresentAndEqual(clusterProxyAddon.Status.Conditions, addonv1alpha1.ManagedClusterAddOnConditionAvailable,
				metav1.ConditionTrue) {
				return fmt.Errorf("cluster proxy addon is not available")
			}
			msaAddon, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(defaultManagedCluster).Get(context.TODO(), "managed-serviceaccount", metav1.GetOptions{})
			if err != nil {
				return err
			}
			if !meta.IsStatusConditionPresentAndEqual(msaAddon.Status.Conditions, addonv1alpha1.ManagedClusterAddOnConditionAvailable,
				metav1.ConditionTrue) {
				return fmt.Errorf("msa addon is not available")
			}
			return nil
		}, 120*time.Second, 2*time.Second).ShouldNot(gomega.HaveOccurred())

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
		// check the ManagedClusterInfo status
		gomega.Eventually(func() error {
			managedClusterInfo, err := util.GetResource(dynamicClient, util.ClusterInfoGVR, defaultManagedCluster, defaultManagedCluster)
			if err != nil {
				return err
			}
			if !util.GetConditionTypeFromStatus(managedClusterInfo, clusterinfov1beta1.ManagedClusterInfoSynced) {
				return fmt.Errorf("the condition of managedClusterInfo is not synced")
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// case1: get logs successfully
		gomega.Eventually(func() error {
			req := restClient.Get().Namespace(defaultManagedCluster).
				Name(defaultManagedCluster).
				Resource("clusterstatuses").
				SubResource("log").Suffix(podNamespace, podName, containerName)

			_, err := req.DoRaw(context.TODO())
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})
})
