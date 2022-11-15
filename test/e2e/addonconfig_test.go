package e2e

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
)

var _ = ginkgo.Describe("Testing work-manager add-on with AddonDeploymentConfigs", func() {
	ginkgo.It("should add node placement for work-manager add-on successfully", func() {
		deployConfigName := "deploy-config"
		addOnName := "work-manager"
		nodeSelector := map[string]string{"kubernetes.io/os": "linux"}
		tolerations := []corev1.Toleration{{Key: "node-role.kubernetes.io/infra", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule}}

		ginkgo.By("Prepare a AddOnDeploymentConfig for work-manager add-on")
		gomega.Eventually(func() error {
			_, err := addonClient.AddonV1alpha1().AddOnDeploymentConfigs(defaultManagedCluster).Get(context.Background(), deployConfigName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				_, err := addonClient.AddonV1alpha1().AddOnDeploymentConfigs(defaultManagedCluster).Create(
					context.Background(),
					&addonapiv1alpha1.AddOnDeploymentConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      deployConfigName,
							Namespace: defaultManagedCluster,
						},
						Spec: addonapiv1alpha1.AddOnDeploymentConfigSpec{
							NodePlacement: &addonapiv1alpha1.NodePlacement{
								NodeSelector: nodeSelector,
								Tolerations:  tolerations,
							},
						},
					},
					metav1.CreateOptions{},
				)
				return err
			}

			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("Add the config to work-manager add-on")
		gomega.Eventually(func() error {
			addon, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(defaultManagedCluster).Get(context.Background(), addOnName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			newAddon := addon.DeepCopy()
			newAddon.Spec.Configs = []addonapiv1alpha1.AddOnConfig{
				{
					ConfigGroupResource: addonapiv1alpha1.ConfigGroupResource{
						Group:    "addon.open-cluster-management.io",
						Resource: "addondeploymentconfigs",
					},
					ConfigReferent: addonapiv1alpha1.ConfigReferent{
						Namespace: defaultManagedCluster,
						Name:      deployConfigName,
					},
				},
			}
			_, err = addonClient.AddonV1alpha1().ManagedClusterAddOns(defaultManagedCluster).Update(context.Background(), newAddon, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("Make sure addon config is referenced")
		gomega.Eventually(func() error {
			addon, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(defaultManagedCluster).Get(context.Background(), addOnName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if len(addon.Status.ConfigReferences) == 0 {
				return fmt.Errorf("no config references in addon status")
			}
			if addon.Status.ConfigReferences[0].Name != deployConfigName {
				return fmt.Errorf("unexpected config references %v", addon.Status.ConfigReferences)
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("Make sure work-manager add-on is configured")
		gomega.Eventually(func() error {
			agentDeploy, err := kubeClient.AppsV1().Deployments("open-cluster-management-agent-addon").Get(context.TODO(), "klusterlet-addon-workmgr", metav1.GetOptions{})
			if err != nil {
				return err
			}

			if !equality.Semantic.DeepEqual(agentDeploy.Spec.Template.Spec.NodeSelector, nodeSelector) {
				return fmt.Errorf("unexpected nodeSeletcor %v", agentDeploy.Spec.Template.Spec.NodeSelector)
			}

			if !equality.Semantic.DeepEqual(agentDeploy.Spec.Template.Spec.Tolerations, tolerations) {
				return fmt.Errorf("unexpected tolerations %v", agentDeploy.Spec.Template.Spec.Tolerations)
			}

			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})
})
