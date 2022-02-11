package agent

import (
	"fmt"

	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/runtime"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

// AgentAddon is a mandatory interface for implementing a custom addon.
// The addon is expected to be registered into an AddonManager so the manager will be invoking the addon
// implementation below as callbacks upon:
//   - receiving valid watch events from ManagedClusterAddon.
//   - receiving valid watch events from ManifestWork.
type AgentAddon interface {

	// Manifests returns a list of manifest resources to be deployed on the managed cluster for this addon.
	// The resources in this list are required to explicitly clarify their TypeMta (i.e. apiVersion, kind)
	// otherwise the addon deploying will fail constantly. A recommended set of addon component returned
	// here will be:
	//   - the hosting namespace of the addon-agents.
	//   - a deployment of the addon agents.
	//   - the configurations (e.g. configmaps) mounted by the deployement.
	//   - the RBAC permission bond to the addon agents *in the managed cluster*. (the hub cluster's RBAC
	//     setup shall be done at GetAgentAddonOptions below.)
	// NB for dispatching namespaced resources, it's recommended to include the namespace in the list.
	Manifests(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) ([]runtime.Object, error)

	// GetAgentAddonOptions returns the agent options for advanced agent customization.
	// A minimal option is merely setting a unique addon name in the AgentAddonOptions.
	GetAgentAddonOptions() AgentAddonOptions
}

// AgentAddonOptions prescribes the future customization for the addon.
type AgentAddonOptions struct {
	// AddonName is the name of the addon.
	// Should be globally unique.
	// +required
	AddonName string

	// Registration prescribes the custom behavior during CSR applying, approval and signing.
	// +optional
	Registration *RegistrationOption

	// InstallStrategy defines that addon should be created in which clusters.
	// Addon will not be installed automatically until a ManagedClusterAddon is applied to the cluster's
	// namespace if InstallStrategy is nil.
	// +optional
	InstallStrategy *InstallStrategy

	// HealthProber defines how is the healthiness status of the ManagedClusterAddon probed.
	// Note that the prescribed prober type here only applies to the automatically installed
	// addons configured via InstallStrategy.
	// If nil, will be defaulted to "Lease" type.
	// +optional
	HealthProber *HealthProber
}

type CSRSignerFunc func(csr *certificatesv1.CertificateSigningRequest) []byte

type CSRApproveFunc func(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn, csr *certificatesv1.CertificateSigningRequest) bool

// RegistrationOption defines how agent is registered to the hub cluster. It needs to define:
// 1. csr with what subject/signer should be created
// 2. how csr is approved
// 3. the RBAC setting of agent on the hub
// 4. how csr is signed if the customized signer is used.
type RegistrationOption struct {
	// CSRConfigurations returns a list of csr configuration for the adddon agent in a managed cluster.
	// A csr will be created from the managed cluster for addon agent with each CSRConfiguration.
	// +required
	CSRConfigurations func(cluster *clusterv1.ManagedCluster) []addonapiv1alpha1.RegistrationConfig

	// CSRApproveCheck checks whether the addon agent registration should be approved by the hub.
	// Addon hub controller can implement this func to auto-approve all the CSRs. A better CSR check is
	// recommended to include (1) the validity of requester's requesting identity and (2) the other request
	// payload such as key-usages.
	// If the function is not set, the registration and certificate renewal of addon agent needs to be approved
	// manually on hub.
	// NB auto-approving csr requires the addon manager to have sufficient RBAC permission for the target
	// signer, e.g.:
	// >>  {    "apiGroups":["certificates.k8s.io"],
	// >>		"resources":["signers"],
	// >>		"resourceNames":["kubernetes.io/kube-apiserver-client"]...}
	// +optional
	CSRApproveCheck CSRApproveFunc

	// PermissionConfig defines the function for an addon to setup rbac permission. This callback doesn't
	// couple with any concrete RBAC Api so the implementation is expected to ensure the RBAC in the hub
	// cluster by calling the kubernetes api explicitly. Additionally we can also extend arbitrary third-party
	// permission setup in this callback.
	// +optional
	PermissionConfig func(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) error

	// CSRSign signs a csr and returns a certificate. It is used when the addon has its own customized signer.
	// The returned byte array shall be a valid non-nil PEM encoded x509 certificate.
	// +optional
	CSRSign CSRSignerFunc
}

type StrategyType string

const InstallAll StrategyType = "*"

// InstallStrategy is the installation strategy of the manifests prescribed by Manifests(..).
type InstallStrategy struct {
	// Type is the type of the installation strategy.
	// Currently supported values: "InstallAll".
	Type StrategyType
	// InstallNamespace is target deploying namespace in the managed cluster upon automatic addon installation.
	InstallNamespace string
}

type HealthProber struct {
	Type HealthProberType
}

type HealthProberType string

const (
	// HealthProberTypeLease indicates the healthiness status will be refreshed, which is
	// leaving the healthiness of ManagedClusterAddon to an empty string.
	HealthProberTypeNone HealthProberType = "None"
	// HealthProberTypeLease indicates the healthiness of the addon is connected with the
	// corresponding lease resource in the cluster namespace with the same name as the addon.
	// Note that the lease object is expected to periodically refresh by a local agent
	// deployed in the managed cluster implementing lease.LeaseUpdater interface.
	HealthProberTypeLease HealthProberType = "Lease"
	// TODO(yue9944882): implement work api health checker
	// HealthProberTypeWork indicates the healthiness of the addon is equal to the overall
	// dispatching status of the corresponding ManifestWork resource.
	// It's applicable to those addons that don't have a local agent instance in the managed
	// clusters.
	//HealthProberTypeWork HealthProberType = "Work"
)

func KubeClientSignerConfigurations(addonName, agentName string) func(cluster *clusterv1.ManagedCluster) []addonapiv1alpha1.RegistrationConfig {
	return func(cluster *clusterv1.ManagedCluster) []addonapiv1alpha1.RegistrationConfig {
		return []addonapiv1alpha1.RegistrationConfig{
			{
				SignerName: certificatesv1.KubeAPIServerClientSignerName,
				Subject: addonapiv1alpha1.Subject{
					User:   DefaultUser(cluster.Name, addonName, agentName),
					Groups: DefaultGroups(cluster.Name, addonName),
				},
			},
		}
	}
}

// DefaultUser returns the default User
func DefaultUser(clusterName, addonName, agentName string) string {
	return fmt.Sprintf("system:open-cluster-management:cluster:%s:addon:%s:agent:%s", clusterName, addonName, agentName)
}

// DefaultGroups returns the default groups
func DefaultGroups(clusterName, addonName string) []string {
	return []string{
		fmt.Sprintf("system:open-cluster-management:cluster:%s:addon:%s", clusterName, addonName),
		fmt.Sprintf("system:open-cluster-management:addon:%s", addonName),
		"system:authenticated",
	}
}

func InstallAllStrategy(installNamespace string) *InstallStrategy {
	return &InstallStrategy{
		Type:             InstallAll,
		InstallNamespace: installNamespace,
	}
}

// ApprovalAllCSRs returns true for all csrs.
func ApprovalAllCSRs(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn, csr *certificatesv1.CertificateSigningRequest) bool {
	return true
}
