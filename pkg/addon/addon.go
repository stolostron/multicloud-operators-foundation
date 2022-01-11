package addon

import (
	"context"
	"embed"
	"fmt"
	"reflect"

	"github.com/openshift/library-go/pkg/assets"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"open-cluster-management.io/addon-framework/pkg/agent"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

var (
	genericScheme = runtime.NewScheme()
	genericCodecs = serializer.NewCodecFactory(genericScheme)
	genericCodec  = genericCodecs.UniversalDeserializer()
)

func init() {
	scheme.AddToScheme(genericScheme)
}

const (
	clusterRoleName = "managed-cluster-workmgr"
	roleBindingName = "managed-cluster-workmgr"
)

var agentDeploymentFiles = []string{
	"manifests/clusterrole.yaml",
	"manifests/clusterrolebinding.yaml",
	"manifests/deployment.yaml",
	"manifests/service.yaml",
	"manifests/sa.yaml",
}

//go:embed manifests
var manifestFiles embed.FS

type foundationAgent struct {
	kubeClient       kubernetes.Interface
	addonName        string
	imageName        string
	installNamespace string
}

func NewAgent(kubeClient kubernetes.Interface, addonName, imageName, installNamespace string) *foundationAgent {
	return &foundationAgent{
		kubeClient:       kubeClient,
		addonName:        addonName,
		imageName:        imageName,
		installNamespace: installNamespace,
	}
}

func (f *foundationAgent) Manifests(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) ([]runtime.Object, error) {
	objects := []runtime.Object{}
	// if the installation namespace is not set, to keep consistent with addon-framework,
	// using open-cluster-management-agent-addon namespace as default namespace.
	installNamespace := addon.Spec.InstallNamespace
	if len(installNamespace) == 0 {
		installNamespace = f.installNamespace
	}

	manifestConfig := struct {
		KubeConfigSecret string
		ClusterName      string
		Namespace        string
		Image            string
	}{
		KubeConfigSecret: fmt.Sprintf("%s-hub-kubeconfig", f.GetAgentAddonOptions().AddonName),
		Namespace:        installNamespace,
		ClusterName:      cluster.Name,
		Image:            f.imageName,
	}

	for _, file := range agentDeploymentFiles {
		template, err := manifestFiles.ReadFile(file)
		if err != nil {
			return objects, err
		}
		raw := assets.MustCreateAssetFromTemplate(file, template, &manifestConfig).Data
		object, _, err := genericCodec.Decode(raw, nil, nil)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func (f *foundationAgent) GetAgentAddonOptions() agent.AgentAddonOptions {
	return agent.AgentAddonOptions{
		AddonName: f.addonName,
		Registration: &agent.RegistrationOption{
			CSRConfigurations: agent.KubeClientSignerConfigurations(f.addonName, f.addonName),
			CSRApproveCheck: func(
				cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn, csr *certificatesv1.CertificateSigningRequest) bool {
				// TODO add more csr check here.
				return true
			},
			PermissionConfig: func(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) error {
				return f.createOrUpdateRoleBinding(cluster.Name)
			},
		},
		InstallStrategy: agent.InstallAllStrategy(f.installNamespace),
	}
}

// createOrUpdateRoleBinding create or update a role binding for a given cluster
func (f *foundationAgent) createOrUpdateRoleBinding(clusterName string) error {
	groups := agent.DefaultGroups(clusterName, f.addonName)
	acmRoleBinding := helpers.NewRoleBindingForClusterRole(clusterRoleName, clusterName).Groups(groups[0]).BindingOrDie()

	// role and rolebinding have the same name
	binding, err := f.kubeClient.RbacV1().RoleBindings(clusterName).Get(context.TODO(), roleBindingName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = f.kubeClient.RbacV1().RoleBindings(clusterName).Create(context.TODO(), &acmRoleBinding, metav1.CreateOptions{})
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
		_, err = f.kubeClient.RbacV1().RoleBindings(clusterName).Update(context.TODO(), binding, metav1.UpdateOptions{})
		return err
	}

	return nil
}
