package e2e

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var viewGVR = schema.GroupVersionResource{
	Group:    "view.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "managedclusterviews",
}

var _ = ginkgo.Describe("Testing ManagedClusterView if agent is ok", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	ginkgo.Context("Creating a managedClusterView", func() {
		ginkgo.It("Should create successfully", func() {
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterViewTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(obj.Object, defaultManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			// create managedClusterView to real cluster
			obj, err = util.CreateResource(dynamicClient, viewGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", viewGVR.Resource)

			ginkgo.By("should get successfully")
			exists, err := util.HasResource(dynamicClient, viewGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())

			ginkgo.By("should have a valid condition")
			gomega.Eventually(func() (interface{}, error) {
				managedClusterView, err := util.GetResource(dynamicClient, viewGVR, defaultManagedCluster, obj.GetName())
				if err != nil {
					return "", err
				}
				// check the managedClusterView status
				condition, err := util.GetConditionFromStatus(managedClusterView)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal("Processing"))
		})
	})

})

var _ = ginkgo.Describe("Testing ManagedClusterView if agent is lost", func() {
	var (
		lostManagedCluster = util.RandomName()
		obj                *unstructured.Unstructured
		err                error
	)

	ginkgo.BeforeEach(func() {
		err = util.ImportManagedCluster(clusterClient, lostManagedCluster)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			_, err = addonClient.AddonV1alpha1().ManagedClusterAddOns(lostManagedCluster).Get(context.Background(), "work-manager", metav1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		err = util.CleanManagedCluster(clusterClient, lostManagedCluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.Context("Creating a managedClusterView", func() {
		ginkgo.It("Should create successfully", func() {
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterViewTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(obj.Object, lostManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			// create managedClusterView to real cluster
			obj, err = util.CreateResource(dynamicClient, viewGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", viewGVR.Resource)

			ginkgo.By("should get successfully")
			exists, err := util.HasResource(dynamicClient, viewGVR, lostManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())

			ginkgo.By("should have a valid condition")
			gomega.Eventually(func() (interface{}, error) {
				managedClusterView, err := util.GetResource(dynamicClient, viewGVR, lostManagedCluster, obj.GetName())
				if err != nil {
					return "", err
				}
				// check the managedClusterView status
				condition, err := util.GetConditionFromStatus(managedClusterView)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal(""))
		})
	})
})

var _ = ginkgo.Describe("Test ManagedClusterView to map a new applied CRD", func() {
	ginkgo.It("Create an example CR, and use managedclusterview to get it", func() {
		var err error
		// create an example CRD
		_, err = apixClient.CustomResourceDefinitions().Create(context.TODO(), &apixv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "examplecrds.example.com",
			},
			Spec: apixv1.CustomResourceDefinitionSpec{
				Group: "example.com",
				Versions: []apixv1.CustomResourceDefinitionVersion{{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &apixv1.CustomResourceValidation{
						OpenAPIV3Schema: &apixv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apixv1.JSONSchemaProps{
								"name": {
									Type: "string",
								},
							},
						},
					},
				}},
				Scope: apixv1.NamespaceScoped,
				Names: apixv1.CustomResourceDefinitionNames{
					Kind:     "ExampleCRD",
					Plural:   "examplecrds",
					Singular: "examplecrd",
				},
			},
		}, metav1.CreateOptions{})
		gomega.Expect(err).To(gomega.BeNil())

		// create example CR
		runtimeExampleGVR := schema.GroupVersionResource{
			Group:    "example.com",
			Version:  "v1",
			Resource: "examplecrds",
		}
		gomega.Eventually(func() error {
			_, err = dynamicClient.Resource(runtimeExampleGVR).Namespace("default").Create(context.TODO(), &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "ExampleCRD",
					"apiVersion": runtimeExampleGVR.Group + "/v1",
					"metadata": map[string]interface{}{
						"name":      "example-crd",
						"namespace": "default",
					},
				},
			}, metav1.CreateOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.Succeed())

		// create mcv with only resource for example
		runtimeMCVGVR := schema.GroupVersionResource{
			Group:    "view.open-cluster-management.io",
			Version:  "v1beta1",
			Resource: "managedclusterviews",
		}
		gomega.Eventually(func() error {
			_, err = dynamicClient.Resource(runtimeMCVGVR).Namespace(defaultManagedCluster).Create(context.TODO(), &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "ManagedClusterView",
					"apiVersion": runtimeMCVGVR.Group + "/v1beta1",
					"metadata": map[string]interface{}{
						"name":      "example-mcv",
						"namespace": defaultManagedCluster,
					},
					"spec": map[string]interface{}{
						"scope": map[string]interface{}{
							"resource":  "examplecrd",
							"name":      "example-crd",
							"namespace": "default",
						},
					},
				},
			}, metav1.CreateOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.Succeed())

		// eventually valid
		gomega.Eventually(func() (interface{}, error) {
			managedClusterView, err := util.GetResource(dynamicClient, viewGVR, defaultManagedCluster,
				"example-mcv")
			if err != nil {
				return "", err
			}
			// check the managedClusterView status
			condition, err := util.GetConditionFromStatus(managedClusterView)
			if err != nil {
				return "", err
			}

			if condition == nil {
				return "", nil
			}
			return condition["type"], nil
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal("Processing"))
	})
})
