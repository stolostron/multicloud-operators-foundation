package e2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var leaseGVR = schema.GroupVersionResource{
	Group:    "coordination.k8s.io",
	Version:  "v1",
	Resource: "leases",
}
var secretGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "secrets",
}
var podGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "pods",
}

const (
	podNamespace = "open-cluster-management-agent"
	secretName   = "hub-kubeconfig-secret"
)

var _ = ginkgo.Describe("Testing Lease", func() {
	ginkgo.Context("Get Lease", func() {
		ginkgo.It("should get/update lease successfully in cluster", func() {
			var firstLeaseTime string
			gomega.Eventually(func() bool {
				lease, err := util.GetResource(dynamicClient, leaseGVR, managedClusterName, "work-manager")
				if err != nil {
					return false
				}
				var found bool
				firstLeaseTime, found, err = unstructured.NestedString(lease.Object, "spec", "renewTime")
				if err != nil || !found {
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				}
				return true
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
			gomega.Eventually(func() bool {
				updatedLease, err := util.GetResource(dynamicClient, leaseGVR, managedClusterName, "work-manager")
				if err != nil {
					return false
				}
				updatedLeaseTime, found, err := unstructured.NestedString(updatedLease.Object, "spec", "renewTime")
				if err != nil || !found {
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				}
				if updatedLeaseTime != firstLeaseTime {
					return true
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})
	})
	ginkgo.Context("Update kubeconfig", func() {
		ginkgo.It("should delete agent pods successfully", func() {
			hubSecret, err := util.GetResource(dynamicClient, secretGVR, podNamespace, secretName)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			podsList, err := util.ListResource(dynamicClient, podGVR, podNamespace, "app=work-manager")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			oriKubeconfig, _, err := unstructured.NestedString(hubSecret.Object, "data", "kubeconfig")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			//update kubeconfig
			err = unstructured.SetNestedField(hubSecret.Object, "Ymhyc2Y=", "data", "kubeconfig")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			_, err = util.UpdateResource(dynamicClient, secretGVR, hubSecret)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			//check pod deleted
			gomega.Eventually(func() bool {
				for _, podItem := range podsList {
					podName, _, err := unstructured.NestedString(podItem.Object, "metadata", "name")
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
					exist, _ := util.HasResource(dynamicClient, podGVR, podNamespace, podName)
					if exist {
						return false
					}
				}
				return true
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//update kubeconfig to right value
			err = unstructured.SetNestedField(hubSecret.Object, oriKubeconfig, "data", "kubeconfig")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			_, err = util.UpdateResource(dynamicClient, secretGVR, hubSecret)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		})
	})
})
