package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HiveConfigSpec defines the desired state of Hive
type HiveConfigSpec struct {
	// ManagedDomains is the list of DNS domains that are managed by the Hive cluster
	// When specifying 'managedDNS: true' in a ClusterDeployment, the ClusterDeployment's
	// baseDomain should be a direct child of one of these domains, otherwise the
	// ClusterDeployment creation will result in a validation error.
	// +optional
	ManagedDomains []ManageDNSConfig `json:"managedDomains,omitempty"`

	// AdditionalCertificateAuthoritiesSecretRef is a list of references to secrets in the
	// 'hive' namespace that contain an additional Certificate Authority to use when communicating
	// with target clusters. These certificate authorities will be used in addition to any self-signed
	// CA generated by each cluster on installation.
	// +optional
	AdditionalCertificateAuthoritiesSecretRef []corev1.LocalObjectReference `json:"additionalCertificateAuthoritiesSecretRef,omitempty"`

	// GlobalPullSecretRef is used to specify a pull secret that will be used globally by all of the cluster deployments.
	// For each cluster deployment, the contents of GlobalPullSecret will be merged with the specific pull secret for
	// a cluster deployment(if specified), with precedence given to the contents of the pull secret for the cluster deployment.
	// +optional
	GlobalPullSecretRef *corev1.LocalObjectReference `json:"globalPullSecretRef,omitempty"`

	// Backup specifies configuration for backup integration.
	// If absent, backup integration will be disabled.
	// +optional
	Backup BackupConfig `json:"backup,omitempty"`

	// FailedProvisionConfig is used to configure settings related to handling provision failures.
	FailedProvisionConfig FailedProvisionConfig `json:"failedProvisionConfig"`

	// LogLevel is the level of logging to use for the Hive controllers.
	// Acceptable levels, from coarsest to finest, are panic, fatal, error, warn, info, debug, and trace.
	// The default level is info.
	// +optional
	LogLevel string `json:"logLevel,omitempty"`

	// SyncSetReapplyInterval is a string duration indicating how much time must pass before SyncSet resources
	// will be reapplied.
	// The default reapply interval is two hours.
	SyncSetReapplyInterval string `json:"syncSetReapplyInterval,omitempty"`

	// HiveAPIEnabled is a boolean controlling whether or not the Hive operator will start up
	// the v1alpha1 aggregated API server.
	HiveAPIEnabled bool `json:"hiveAPIEnabled,omitempty"`

	// MaintenanceMode can be set to true to disable the hive controllers in situations where we need to ensure
	// nothing is running that will add or act upon finalizers on Hive types. This should rarely be needed.
	// Sets replicas to 0 for the hive-controllers deployment to accomplish this.
	MaintenanceMode *bool `json:"maintenanceMode,omitempty"`
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
}

// FailedProvisionConfig contains settings to control behavior undertaken by Hive when an installation attempt fails.
type FailedProvisionConfig struct {

	// SkipGatherLogs disables functionality that attempts to gather full logs from the cluster if an installation
	// fails for any reason. The logs will be stored in a persistent volume for up to 7 days.
	SkipGatherLogs bool `json:"skipGatherLogs,omitempty"`
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

	// As other cloud providers are supported, additional fields will be
	// added for each of those cloud providers. Only a single cloud provider
	// may be configured at a time.
}

// ManageDNSAWSConfig contains AWS-specific info to manage a given domain.
type ManageDNSAWSConfig struct {
	// CredentialsSecretRef references a secret that will be used to authenticate with
	// AWS Route53. It will need permission to manage entries for the domain
	// listed in the parent ManageDNSConfig object.
	// Secret should have AWS keys named 'aws_access_key_id' and 'aws_secret_access_key'.
	// +optional
	CredentialsSecretRef corev1.LocalObjectReference `json:"credentialsSecretRef,omitempty"`

	// Region is the AWS region to use for route53 operations.
	// This defaults to us-east-1.
	// For AWS China, use cn-northwest-1.
	// +optional
	Region string `json:"region,omitempty"`
}

// ManageDNSGCPConfig contains GCP-specific info to manage a given domain.
type ManageDNSGCPConfig struct {
	// CredentialsSecretRef references a secret that will be used to authenticate with
	// GCP DNS. It will need permission to manage entries in each of the
	// managed domains for this cluster.
	// listed in the parent ManageDNSConfig object.
	// Secret should have a key named 'osServiceAccount.json'.
	// The credentials must specify the project to use.
	// +optional
	CredentialsSecretRef corev1.LocalObjectReference `json:"credentialsSecretRef,omitempty"`
}

// +genclient:nonNamespaced
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HiveConfig is the Schema for the hives API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
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
