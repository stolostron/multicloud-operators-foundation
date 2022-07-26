package addon

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/stolostron/cluster-lifecycle-api/helpers/imageregistry"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation"
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

	// annotationNodeSelector is key name of nodeSelector annotation on ManagedCluster
	annotationNodeSelector = "open-cluster-management/nodeSelector"

	// annotationValues is key name of tolerations annotation on ManagedCluster
	annotationTolerations = "open-cluster-management/tolerations"
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
	Product                         string              `json:"product,omitempty"`
	Tolerations                     []corev1.Toleration `json:"tolerations,omitempty"`
	GlobalValues                    GlobalValues        `json:"global,omitempty,omitempty"`
	EnableSyncLabelsToClusterClaims bool                `json:"enableSyncLabelsToClusterClaims"`
	EnableNodeCollector             bool                `json:"enableNodeCollector"`
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
		enableNodeCollector := true
		if value, ok := addon.GetAnnotations()[addonconstants.HostingClusterNameAnnotationKey]; ok && value != "" {
			enableSyncLabelsToClusterClaims = false
			enableNodeCollector = false
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
			EnableNodeCollector:             enableNodeCollector,
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

		tolerations, err := getTolerations(cluster)
		if err != nil {
			return nil, err
		}
		if len(tolerations) != 0 {
			addonValues.Tolerations = tolerations
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

	nodeSelectorString, ok := managedCluster.Annotations[annotationNodeSelector]
	if !ok {
		return nodeSelector, nil
	}

	if err := json.Unmarshal([]byte(nodeSelectorString), &nodeSelector); err != nil {
		return nil, fmt.Errorf("invalid nodeSelector annotation of cluster %s, %v", managedCluster.Name, err)
	}

	if err := validateNodeSelector(nodeSelector); err != nil {
		return nil, fmt.Errorf("invalid nodeSelector annotation of cluster %s, %v", managedCluster.Name, err)
	}

	return nodeSelector, nil
}

// refer to https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation.go#L3498
func validateNodeSelector(nodeSelector map[string]string) error {
	errs := []error{}
	for key, val := range nodeSelector {
		if errMsgs := validation.IsQualifiedName(key); len(errMsgs) != 0 {
			errs = append(errs, fmt.Errorf(strings.Join(errMsgs, ";")))
		}
		if errMsgs := validation.IsValidLabelValue(val); len(errMsgs) != 0 {
			errs = append(errs, fmt.Errorf(strings.Join(errMsgs, ";")))
		}
	}
	return utilerrors.NewAggregate(errs)
}

func getTolerations(cluster *clusterv1.ManagedCluster) ([]corev1.Toleration, error) {
	tolerations := []corev1.Toleration{}

	tolerationsString, ok := cluster.Annotations[annotationTolerations]
	if !ok {
		return tolerations, nil
	}

	if err := json.Unmarshal([]byte(tolerationsString), &tolerations); err != nil {
		return nil, fmt.Errorf("invalid tolerations annotation of cluster %s, %v", cluster.Name, err)
	}

	if err := validateTolerations(tolerations); err != nil {
		return nil, fmt.Errorf("invalid tolerations annotation of cluster %s, %v", cluster.Name, err)
	}

	return tolerations, nil
}

// refer to https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation.go#L3330
func validateTolerations(tolerations []corev1.Toleration) error {
	errs := []error{}
	for _, toleration := range tolerations {
		// validate the toleration key
		if len(toleration.Key) > 0 {
			if errMsgs := validation.IsQualifiedName(toleration.Key); len(errMsgs) != 0 {
				errs = append(errs, fmt.Errorf(strings.Join(errMsgs, ";")))
			}
		}

		// empty toleration key with Exists operator and empty value means match all taints
		if len(toleration.Key) == 0 && toleration.Operator != corev1.TolerationOpExists {
			if len(toleration.Operator) == 0 {
				errs = append(errs, fmt.Errorf(
					"operator must be Exists when `key` is empty, which means \"match all values and all keys\""))
			}
		}

		if toleration.TolerationSeconds != nil && toleration.Effect != corev1.TaintEffectNoExecute {
			errs = append(errs, fmt.Errorf("effect must be 'NoExecute' when `tolerationSeconds` is set"))
		}

		// validate toleration operator and value
		switch toleration.Operator {
		// empty operator means Equal
		case corev1.TolerationOpEqual, "":
			if errMsgs := validation.IsValidLabelValue(toleration.Value); len(errMsgs) != 0 {
				errs = append(errs, fmt.Errorf(strings.Join(errMsgs, ";")))
			}
		case corev1.TolerationOpExists:
			if len(toleration.Value) > 0 {
				errs = append(errs, fmt.Errorf("value must be empty when `operator` is 'Exists'"))
			}
		default:
			errs = append(errs, fmt.Errorf("the operator %q is not supported", toleration.Operator))
		}

		// validate toleration effect, empty toleration effect means match all taint effects
		if len(toleration.Effect) > 0 {
			switch toleration.Effect {
			case corev1.TaintEffectNoSchedule, corev1.TaintEffectPreferNoSchedule, corev1.TaintEffectNoExecute:
				// allowed values are NoSchedule, PreferNoSchedule and NoExecute
			default:
				errs = append(errs, fmt.Errorf("the effect %q is not supported", toleration.Effect))
			}
		}
	}

	return utilerrors.NewAggregate(errs)
}
