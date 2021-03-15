package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// FeatureGateAgentInstallStrategy enables the use of the alpha ClusterDeployment agent based
	// install strategy and platforms.
	FeatureGateAgentInstallStrategy = "AlphaAgentInstallStrategy"
)

// HiveConfigSpec defines the desired state of Hive
type HiveConfigSpec struct {

	// TargetNamespace is the namespace where the core Hive components should be run. Defaults to "hive". Will be
	// created if it does not already exist. All resource references in HiveConfig can be assumed to be in the
	// TargetNamespace.
	// +optional
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// ManagedDomains is the list of DNS domains that are managed by the Hive cluster
	// When specifying 'manageDNS: true' in a ClusterDeployment, the ClusterDeployment's
	// baseDomain should be a direct child of one of these domains, otherwise the
	// ClusterDeployment creation will result in a validation error.
	// +optional
	ManagedDomains []ManageDNSConfig `json:"managedDomains,omitempty"`

	// AdditionalCertificateAuthoritiesSecretRef is a list of references to secrets in the
	// TargetNamespace that contain an additional Certificate Authority to use when communicating
	// with target clusters. These certificate authorities will be used in addition to any self-signed
	// CA generated by each cluster on installation.
	// +optional
	AdditionalCertificateAuthoritiesSecretRef []corev1.LocalObjectReference `json:"additionalCertificateAuthoritiesSecretRef,omitempty"`

	// GlobalPullSecretRef is used to specify a pull secret that will be used globally by all of the cluster deployments.
	// For each cluster deployment, the contents of GlobalPullSecret will be merged with the specific pull secret for
	// a cluster deployment(if specified), with precedence given to the contents of the pull secret for the cluster deployment.
	// The global pull secret is assumed to be in the TargetNamespace.
	// +optional
	GlobalPullSecretRef *corev1.LocalObjectReference `json:"globalPullSecretRef,omitempty"`

	// Backup specifies configuration for backup integration.
	// If absent, backup integration will be disabled.
	// +optional
	Backup BackupConfig `json:"backup,omitempty"`

	// FailedProvisionConfig is used to configure settings related to handling provision failures.
	// +optional
	FailedProvisionConfig FailedProvisionConfig `json:"failedProvisionConfig,omitempty"`

	// LogLevel is the level of logging to use for the Hive controllers.
	// Acceptable levels, from coarsest to finest, are panic, fatal, error, warn, info, debug, and trace.
	// The default level is info.
	// +optional
	LogLevel string `json:"logLevel,omitempty"`

	// SyncSetReapplyInterval is a string duration indicating how much time must pass before SyncSet resources
	// will be reapplied.
	// The default reapply interval is two hours.
	SyncSetReapplyInterval string `json:"syncSetReapplyInterval,omitempty"`

	// MaintenanceMode can be set to true to disable the hive controllers in situations where we need to ensure
	// nothing is running that will add or act upon finalizers on Hive types. This should rarely be needed.
	// Sets replicas to 0 for the hive-controllers deployment to accomplish this.
	MaintenanceMode *bool `json:"maintenanceMode,omitempty"`

	// DeprovisionsDisabled can be set to true to block deprovision jobs from running.
	DeprovisionsDisabled *bool `json:"deprovisionsDisabled,omitempty"`

	// DeleteProtection can be set to "enabled" to turn on automatic delete protection for ClusterDeployments. When
	// enabled, Hive will add the "hive.openshift.io/protected-delete" annotation to new ClusterDeployments. Once a
	// ClusterDeployment has been installed, a user must remove the annotation from a ClusterDeployment prior to
	// deleting it.
	// +kubebuilder:validation:Enum=enabled
	// +optional
	DeleteProtection DeleteProtectionType `json:"deleteProtection,omitempty"`

	// DisabledControllers allows selectively disabling Hive controllers by name.
	// The name of an individual controller matches the name of the controller as seen in the Hive logging output.
	DisabledControllers []string `json:"disabledControllers,omitempty"`

	// ControllersConfig is used to configure different hive controllers
	// +optional
	ControllersConfig *ControllersConfig `json:"controllersConfig,omitempty"`

	FeatureGates *FeatureGateSelection `json:"featureGates,omitempty"`
}

// FeatureSet defines the set of feature gates that should be used.
// +kubebuilder:validation:Enum="";Custom
type FeatureSet string

var (
	// DefaultFeatureSet feature set is the default things supported as part of normal supported platform.
	DefaultFeatureSet FeatureSet = ""

	// CustomFeatureSet allows the enabling or disabling of any feature. Turning this feature set on IS NOT SUPPORTED.
	// Because of its nature, this setting cannot be validated.  If you have any typos or accidentally apply invalid combinations
	// it might leave object in a state that is unrecoverable.
	CustomFeatureSet FeatureSet = "Custom"
)

// FeatureGateSelection allows selecting feature gates for the controller.
type FeatureGateSelection struct {
	// featureSet changes the list of features in the cluster.  The default is empty.  Be very careful adjusting this setting.
	// +unionDiscriminator
	// +optional
	FeatureSet FeatureSet `json:"featureSet,omitempty"`

	// custom allows the enabling or disabling of any feature.
	// Because of its nature, this setting cannot be validated.  If you have any typos or accidentally apply invalid combinations
	// might cause unknown behavior. featureSet must equal "Custom" must be set to use this field.
	// +optional
	// +nullable
	Custom *FeatureGatesEnabled `json:"custom,omitempty"`
}

// FeatureGatesEnabled is list of feature gates that must be enabled.
type FeatureGatesEnabled struct {
	// enabled is a list of all feature gates that you want to force on
	// +optional
	Enabled []string `json:"enabled,omitempty"`
}

// FeatureSets Contains a map of Feature names to Enabled/Disabled Feature.
var FeatureSets = map[FeatureSet]*FeatureGatesEnabled{
	DefaultFeatureSet: {
		Enabled: []string{},
	},
	CustomFeatureSet: {
		Enabled: []string{},
	},
}

// HiveConfigStatus defines the observed state of Hive
type HiveConfigStatus struct {
	// AggregatorClientCAHash keeps an md5 hash of the aggregator client CA
	// configmap data from the openshift-config-managed namespace. When the configmap changes,
	// admission is redeployed.
	AggregatorClientCAHash string `json:"aggregatorClientCAHash,omitempty"`

	// ObservedGeneration will record the most recently processed HiveConfig object's generation.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConfigApplied will be set by the hive operator to indicate whether or not the LastGenerationObserved
	// was successfully reconciled.
	ConfigApplied bool `json:"configApplied,omitempty"`
}

// BackupConfig contains settings for the Velero backup integration.
type BackupConfig struct {
	// Velero specifies configuration for the Velero backup integration.
	// +optional
	Velero VeleroBackupConfig `json:"velero,omitempty"`

	// MinBackupPeriodSeconds specifies that a minimum of MinBackupPeriodSeconds will occur in between each backup.
	// This is used to rate limit backups. This potentially batches together multiple changes into 1 backup.
	// No backups will be lost as changes that happen during this interval are queued up and will result in a
	// backup happening once the interval has been completed.
	// +optional
	MinBackupPeriodSeconds *int `json:"minBackupPeriodSeconds,omitempty"`
}

// VeleroBackupConfig contains settings for the Velero backup integration.
type VeleroBackupConfig struct {
	// Enabled dictates if Velero backup integration is enabled.
	// If not specified, the default is disabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Namespace specifies in which namespace velero backup objects should be created.
	// If not specified, the default is a namespace named "velero".
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// FailedProvisionConfig contains settings to control behavior undertaken by Hive when an installation attempt fails.
type FailedProvisionConfig struct {

	// TODO: Figure out how to mark SkipGatherLogs as deprecated (more than just a comment)

	// DEPRECATED: This flag is no longer respected and will be removed in the future.
	SkipGatherLogs bool                      `json:"skipGatherLogs,omitempty"`
	AWS            *FailedProvisionAWSConfig `json:"aws,omitempty"`
}

// ManageDNSConfig contains the domain being managed, and the cloud-specific
// details for accessing/managing the domain.
type ManageDNSConfig struct {

	// Domains is the list of domains that hive will be managing entries for with the provided credentials.
	Domains []string `json:"domains"`

	// AWS contains AWS-specific settings for external DNS
	// +optional
	AWS *ManageDNSAWSConfig `json:"aws,omitempty"`

	// GCP contains GCP-specific settings for external DNS
	// +optional
	GCP *ManageDNSGCPConfig `json:"gcp,omitempty"`

	// Azure contains Azure-specific settings for external DNS
	// +optional
	Azure *ManageDNSAzureConfig `json:"azure,omitempty"`

	// As other cloud providers are supported, additional fields will be
	// added for each of those cloud providers. Only a single cloud provider
	// may be configured at a time.
}

// FailedProvisionAWSConfig contains AWS-specific info to upload log files.
type FailedProvisionAWSConfig struct {
	// CredentialsSecretRef references a secret in the TargetNamespace that will be used to authenticate with
	// AWS S3. It will need permission to upload logs to S3.
	// Secret should have keys named aws_access_key_id and aws_secret_access_key that contain the AWS credentials.
	// Example Secret:
	//   data:
	//     aws_access_key_id: minio
	//     aws_secret_access_key: minio123
	CredentialsSecretRef corev1.LocalObjectReference `json:"credentialsSecretRef"`

	// Region is the AWS region to use for S3 operations.
	// This defaults to us-east-1.
	// For AWS China, use cn-northwest-1.
	// +optional
	Region string `json:"region,omitempty"`

	// ServiceEndpoint is the url to connect to an S3 compatible provider.
	ServiceEndpoint string `json:"serviceEndpoint,omitempty"`

	// Bucket is the S3 bucket to store the logs in.
	Bucket string `json:"bucket,omitempty"`
}

// ManageDNSAWSConfig contains AWS-specific info to manage a given domain.
type ManageDNSAWSConfig struct {
	// CredentialsSecretRef references a secret in the TargetNamespace that will be used to authenticate with
	// AWS Route53. It will need permission to manage entries for the domain
	// listed in the parent ManageDNSConfig object.
	// Secret should have AWS keys named 'aws_access_key_id' and 'aws_secret_access_key'.
	CredentialsSecretRef corev1.LocalObjectReference `json:"credentialsSecretRef"`

	// Region is the AWS region to use for route53 operations.
	// This defaults to us-east-1.
	// For AWS China, use cn-northwest-1.
	// +optional
	Region string `json:"region,omitempty"`
}

// ManageDNSGCPConfig contains GCP-specific info to manage a given domain.
type ManageDNSGCPConfig struct {
	// CredentialsSecretRef references a secret in the TargetNamespace that will be used to authenticate with
	// GCP DNS. It will need permission to manage entries in each of the
	// managed domains for this cluster.
	// listed in the parent ManageDNSConfig object.
	// Secret should have a key named 'osServiceAccount.json'.
	// The credentials must specify the project to use.
	CredentialsSecretRef corev1.LocalObjectReference `json:"credentialsSecretRef"`
}

type DeleteProtectionType string

const (
	DeleteProtectionEnabled DeleteProtectionType = "enabled"
)

// ManageDNSAzureConfig contains Azure-specific info to manage a given domain
type ManageDNSAzureConfig struct {
	// CredentialsSecretRef references a secret in the TargetNamespace that will be used to authenticate with
	// Azure DNS. It wil need permission to manage entries in each of the
	// managed domains listed in the parent ManageDNSConfig object.
	// Secret should have a key named 'osServicePrincipal.json'
	CredentialsSecretRef corev1.LocalObjectReference `json:"credentialsSecretRef"`

	// ResourceGroupName specifies the Azure resource group containing the DNS zones
	// for the domains being managed.
	ResourceGroupName string `json:"resourceGroupName"`
}

// ControllerConfig contains the configuration for a controller
type ControllerConfig struct {
	// ConcurrentReconciles specifies number of concurrent reconciles for a controller
	// +optional
	ConcurrentReconciles *int32 `json:"concurrentReconciles,omitempty"`
	// ClientQPS specifies client rate limiter QPS for a controller
	// +optional
	ClientQPS *int32 `json:"clientQPS,omitempty"`
	// ClientBurst specifies client rate limiter burst for a controller
	// +optional
	ClientBurst *int32 `json:"clientBurst,omitempty"`
	// QueueQPS specifies workqueue rate limiter QPS for a controller
	// +optional
	QueueQPS *int32 `json:"queueQPS,omitempty"`
	// QueueBurst specifies workqueue rate limiter burst for a controller
	// +optional
	QueueBurst *int32 `json:"queueBurst,omitempty"`
	// Replicas specifies the number of replicas the specific controller pod should use.
	// This is ONLY for controllers that have been split out into their own pods.
	// This is ignored for all others.
	Replicas *int32 `json:"replicas,omitempty"`
}

// +kubebuilder:validation:Enum=clusterDeployment;clusterrelocate;clusterstate;clusterversion;controlPlaneCerts;dnsendpoint;dnszone;remoteingress;remotemachineset;syncidentityprovider;unreachable;velerobackup;clusterprovision;clusterDeprovision;clusterpool;clusterpoolnamespace;hibernation;clusterclaim;metrics;clustersync
type ControllerName string

func (controllerName ControllerName) String() string {
	return string(controllerName)
}

// ControllerNames is a slice of controller names
type ControllerNames []ControllerName

// Contains says whether or not the controller name is in the slice of controller names.
func (c ControllerNames) Contains(controllerName ControllerName) bool {
	for _, curControllerName := range c {
		if curControllerName == controllerName {
			return true
		}
	}

	return false
}

// WARNING: All the controller names below should also be added to the kubebuilder validation of the type ControllerName
const (
	ClusterClaimControllerName         ControllerName = "clusterclaim"
	ClusterDeploymentControllerName    ControllerName = "clusterDeployment"
	ClusterDeprovisionControllerName   ControllerName = "clusterDeprovision"
	ClusterpoolControllerName          ControllerName = "clusterpool"
	ClusterpoolNamespaceControllerName ControllerName = "clusterpoolnamespace"
	ClusterProvisionControllerName     ControllerName = "clusterProvision"
	ClusterRelocateControllerName      ControllerName = "clusterRelocate"
	ClusterStateControllerName         ControllerName = "clusterState"
	ClusterVersionControllerName       ControllerName = "clusterversion"
	ControlPlaneCertsControllerName    ControllerName = "controlPlaneCerts"
	DNSEndpointControllerName          ControllerName = "dnsendpoint"
	DNSZoneControllerName              ControllerName = "dnszone"
	HibernationControllerName          ControllerName = "hibernation"
	RemoteIngressControllerName        ControllerName = "remoteingress"
	RemoteMachinesetControllerName     ControllerName = "remotemachineset"
	SyncIdentityProviderControllerName ControllerName = "syncidentityprovider"
	UnreachableControllerName          ControllerName = "unreachable"
	VeleroBackupControllerName         ControllerName = "velerobackup"
	MetricsControllerName              ControllerName = "metrics"
	ClustersyncControllerName          ControllerName = "clustersync"
)

// SpecificControllerConfig contains the configuration for a specific controller
type SpecificControllerConfig struct {
	// Name specifies the name of the controller
	Name ControllerName `json:"name"`
	// ControllerConfig contains the configuration for the controller specified by Name field
	Config ControllerConfig `json:"config"`
}

// ControllersConfig contains default as well as controller specific configurations
type ControllersConfig struct {
	// Default specifies default configuration for all the controllers, can be used to override following coded defaults
	// default for concurrent reconciles is 5
	// default for client qps is 5
	// default for client burst is 10
	// default for queue qps is 10
	// default for queue burst is 100
	// +optional
	Default *ControllerConfig `json:"default,omitempty"`
	// Controllers contains a list of configurations for different controllers
	// +optional
	Controllers []SpecificControllerConfig `json:"controllers,omitempty"`
}

// +genclient:nonNamespaced
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HiveConfig is the Schema for the hives API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type HiveConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HiveConfigSpec   `json:"spec,omitempty"`
	Status HiveConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HiveConfigList contains a list of Hive
type HiveConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HiveConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HiveConfig{}, &HiveConfigList{})
}
