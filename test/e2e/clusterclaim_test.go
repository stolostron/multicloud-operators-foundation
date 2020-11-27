package e2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
)

var clusterClaimGVR = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "clusterclaims",
}

var _ = ginkgo.Describe("Testing ClusterClaim", func() {
	var requiredClaimNames = []string{
		"id.k8s.io",
		"kubeversion.open-cluster-management.io",
	}

	ginkgo.It("should get the required ClusterClaims successfully the managed cluster", func() {
		gomega.Eventually(func() bool {
			for _, claimName := range requiredClaimNames {
				exists, err := util.HasClusterResource(dynamicClient, clusterClaimGVR, claimName)
				if err != nil {
					return false
				}
				if !exists {
					return false
				}
			}

			return true
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
	})

	ginkgo.It("should recreate the ClusterClaim once it is deleted", func() {
		claimName := requiredClaimNames[0]

		// make sure the claim is in place
		gomega.Eventually(func() bool {
			exists, err := util.HasClusterResource(dynamicClient, clusterClaimGVR, claimName)
			if err != nil {
				return false
			}

			return exists
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

		// delete the claim
		err := util.DeleteClusterResource(dynamicClient, clusterClaimGVR, claimName)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// check if it is recreated
		gomega.Eventually(func() bool {
			exists, err := util.HasClusterResource(dynamicClient, clusterClaimGVR, claimName)
			if err != nil {
				return false
			}

			return exists
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
	})

	ginkgo.It("should rollback the change of the ClusterClaim once it is modified", func() {
		claimName := requiredClaimNames[0]

		// get the original claim
		var claim *unstructured.Unstructured
		var err error
		gomega.Eventually(func() bool {
			claim, err = util.GetClusterResource(dynamicClient, clusterClaimGVR, claimName)
			if err != nil {
				return false
			}

			return true
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

		originalValue, _, err := unstructured.NestedString(claim.Object, "Spec", "Value")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// modify the the claim value
		err = unstructured.SetNestedField(claim.Object, rand.String(6), "Spec", "Value")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		_, err = util.UpdateClusterResource(dynamicClient, clusterClaimGVR, claim)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// check if the change is rollback
		gomega.Eventually(func() bool {
			claim, err = util.GetClusterResource(dynamicClient, clusterClaimGVR, claimName)
			if err != nil {
				return false
			}

			value, _, err := unstructured.NestedString(claim.Object, "Spec", "Value")
			if err != nil {
				return false
			}

			return originalValue == value
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
	})
})
