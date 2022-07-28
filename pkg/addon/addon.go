package addon

import (
	"context"
	"embed"
	"encoding/json"
	"reflect"

	"github.com/stolostron/cluster-lifecycle-api/helpers/imageregistry"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"open-cluster-management.io/addon-framework/pkg/addonfactory"
	addonconstants "open-cluster-management.io/addon-framework/pkg/addonmanager/constants"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/utils"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

const (
	WorkManagerAddonName = "work-manager"
	// the clusterRole has been installed with the ocm-controller deployment
	clusterRoleName = "managed-cluster-workmgr"
	roleBindingName = "managed-cluster-workmgr"

	// annotationNodeSelector is key name of nodeSelector annotation synced from mch
	annotationNodeSelector = "open-cluster-management/nodeSelector"
)

//go:embed manifests
//go:embed manifests/chart
//go:embed manifests/chart/templates/_helpers.tpl
var ChartFS embed.FS

const ChartDir = "manifests/chart"

type GlobalValues struct {
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,"`
	ImagePullSecret string            `json:"imagePullSecret"`
	ImageOverrides  map[string]string `json:"imageOverrides,"`
	NodeSelector    map[string]string `json:"nodeSelector,"`
}

type Values struct {
	Product                         string       `json:"product,omitempty"`
	GlobalValues                    GlobalValues `json:"global,omitempty,omitempty"`
	EnableSyncLabelsToClusterClaims bool         `json:"enableSyncLabelsToClusterClaims"`
	EnableNodeCapacity              bool         `json:"enableNodeCapacity"`
}

func NewGetValuesFunc(imageName string) addonfactory.GetValuesFunc {
	return func(cluster *clusterv1.ManagedCluster,
		addon *addonapiv1alpha1.ManagedClusterAddOn) (addonfactory.Values, error) {
		overrideName, err := imageregistry.OverrideImageByAnnotation(cluster.GetAnnotations(), imageName)
		if err != nil {
			return nil, err
		}

		// if addon is hosed mode, the enableSyncLabelsToClusterClaims,enableNodeCollector is false
		enableSyncLabelsToClusterClaims := true
		enableNodeCapacity := true
		if value, ok := addon.GetAnnotations()[addonconstants.HostingClusterNameAnnotationKey]; ok && value != "" {
			enableSyncLabelsToClusterClaims = false
			enableNodeCapacity = false
		}

		addonValues := Values{
			GlobalValues: GlobalValues{
				ImagePullPolicy: corev1.PullIfNotPresent,
				ImagePullSecret: "open-cluster-management-image-pull-credentials",
				ImageOverrides: map[string]string{
					"multicloud_manager": overrideName,
				},
				NodeSelector: map[string]string{},
			},
			EnableSyncLabelsToClusterClaims: enableSyncLabelsToClusterClaims,
			EnableNodeCapacity:              enableNodeCapacity,
		}

		for _, claim := range cluster.Status.ClusterClaims {
			if claim.Name == "product.open-cluster-management.io" {
				addonValues.Product = claim.Value
				break
			}
		}

		nodeSelector, err := getNodeSelector(cluster)
		if err != nil {
			klog.Errorf("failed to get nodeSelector from managedCluster. %v", err)
			return nil, err
		}
		if len(nodeSelector) != 0 {
			addonValues.GlobalValues.NodeSelector = nodeSelector
		}

		values, err := addonfactory.JsonStructToValues(addonValues)
		if err != nil {
			return nil, err
		}
		return values, nil
	}
}

func NewRegistrationOption(kubeClient kubernetes.Interface, addonName string) *agent.RegistrationOption {
	return &agent.RegistrationOption{
		CSRConfigurations: agent.KubeClientSignerConfigurations(addonName, addonName),
		CSRApproveCheck:   utils.DefaultCSRApprover(addonName),
		PermissionConfig: func(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) error {
			return createOrUpdateRoleBinding(kubeClient, addonName, cluster.Name)
		},
	}
}

// createOrUpdateRoleBinding create or update a role binding for a given cluster
func createOrUpdateRoleBinding(kubeClient kubernetes.Interface, addonName, clusterName string) error {
	groups := agent.DefaultGroups(clusterName, addonName)
	acmRoleBinding := helpers.NewRoleBindingForClusterRole(clusterRoleName, clusterName).Groups(groups[0]).BindingOrDie()

	// role and rolebinding have the same name
	binding, err := kubeClient.RbacV1().RoleBindings(clusterName).Get(context.TODO(), roleBindingName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = kubeClient.RbacV1().RoleBindings(clusterName).Create(context.TODO(), &acmRoleBinding, metav1.CreateOptions{})
		}
		return err
	}

	needUpdate := false
	if !reflect.DeepEqual(acmRoleBinding.RoleRef, binding.RoleRef) {
		needUpdate = true
		binding.RoleRef = acmRoleBinding.RoleRef
	}
	if !reflect.DeepEqual(acmRoleBinding.Subjects, binding.Subjects) {
		needUpdate = true
		binding.Subjects = acmRoleBinding.Subjects
	}
	if needUpdate {
		_, err = kubeClient.RbacV1().RoleBindings(clusterName).Update(context.TODO(), binding, metav1.UpdateOptions{})
		return err
	}

	return nil
}

func getNodeSelector(managedCluster *clusterv1.ManagedCluster) (map[string]string, error) {
	nodeSelector := map[string]string{}

	if managedCluster.GetName() == "local-cluster" {
		annotations := managedCluster.GetAnnotations()
		if nodeSelectorString, ok := annotations[annotationNodeSelector]; ok {
			if err := json.Unmarshal([]byte(nodeSelectorString), &nodeSelector); err != nil {
				klog.Error(err, "failed to unmarshal nodeSelector annotation of cluster %v", managedCluster.GetName())
				return nodeSelector, err
			}
		}
	}

	return nodeSelector, nil
}
