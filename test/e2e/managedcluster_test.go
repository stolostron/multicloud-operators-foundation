package e2e

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Testing ManagedCluster", func() {
	ginkgo.Context("Get ManagedCluster cpu worker capacity", func() {
		ginkgo.It("should get a cpu_worker successfully in status of managedcluster", func() {
			gomega.Eventually(func() error {
				cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.Background(), managedClusterName, metav1.GetOptions{})
				if err != nil {
					return err
				}

				capacity := cluster.Status.Capacity
				if _, ok := capacity["cpu_worker"]; !ok {
					return fmt.Errorf("Expect cpu_worker to be set, but got %v", capacity)
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})
	})
})
