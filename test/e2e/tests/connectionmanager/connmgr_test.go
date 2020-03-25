// +build integration

package connmgr_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	eventuallyTimeout  = 60
	eventuallyInterval = 1
)

const (
	configMapNS         = "kube-system"
	configMapName       = "extension-apiserver-authentication"
	apiserverNS         = "multicloud-system"
	apiserverLabel      = "app=mcm-apiserver"
	apiserverDeployName = "mcm-apiserver"
)

var configMapGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "configmaps",
}

var podGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "pods",
}

var (
	dynamicClient dynamic.Interface
)

var _ = BeforeSuite(func() {
	var err error
	dynamicClient, err = common.NewDynamicClient()
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = Describe("ConnectionManager", func() {
	Describe("Reloading mcm-apiserver", func() {
		It("should reload mcm-apiserver successfully", func() {
			cm, err := common.GetResource(dynamicClient, configMapGVR, configMapNS, configMapName)
			Ω(err).ShouldNot(HaveOccurred())
			pods, err := common.ListResource(dynamicClient, podGVR, apiserverNS, apiserverLabel)
			Ω(err).ShouldNot(HaveOccurred())
			apiserverName := pods[0].GetName()

			// change the configmap and wait for reloading the apiserver
			updatingCM := cm.DeepCopy()
			err = unstructured.SetNestedField(updatingCM.Object, "for-test", "data", "client-ca-file")
			Ω(err).ShouldNot(HaveOccurred())
			_, err = dynamicClient.Resource(configMapGVR).Namespace(configMapNS).Update(updatingCM, metav1.UpdateOptions{})
			Ω(err).ShouldNot(HaveOccurred())
			time.Sleep(5 * time.Second)
			Eventually(func() bool {
				pods, err := common.ListResource(dynamicClient, podGVR, apiserverNS, apiserverLabel)
				if err != nil {
					fmt.Fprintf(GinkgoWriter, "failed to list the apiserver pods: %v\n", err)
					return false
				}
				if pods[0].GetName() != apiserverName {
					return true
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
		})

		AfterEach(func() {
			// rollback the configmap and make sure the apiserver is ready
			err := dynamicClient.Resource(configMapGVR).Namespace(configMapNS).Delete(configMapName, &metav1.DeleteOptions{})
			Ω(err).ShouldNot(HaveOccurred())
			Eventually(func() bool {
				_, err := common.GetReadyManagedClusters(dynamicClient)
				if err != nil {
					fmt.Fprintf(GinkgoWriter, "failed to get clusters: %v, retry again\n", err)
					return false
				}
				return true
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
		})
	})
})
