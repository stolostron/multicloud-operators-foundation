package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ManagedClusterResourceNamespace is the namespace on the managed cluster where BareMetalHosts are placed.
const ManagedClusterResourceNamespace string = "openshift-machine-api"

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BMCDetails contains the information necessary to communicate with
// the bare metal controller module on host.
type BMCDetails struct {

	// Address holds the URL for accessing the controller on the
	// network.
	Address string `json:"address"`

	// The name of the secret containing the BMC credentials (requires
	// keys "username" and "password").
	CredentialsName string `json:"credentialsName"`
}

// Role represents the role assigned to the asset
type Role string

const (
	// MasterRole is the master role assigned to the asset
	MasterRole Role = "master"

	// WorkerRole is the worker role assigned to the asset
	WorkerRole Role = "worker"
)

// BareMetalAssetSpec defines the desired state of BareMetalAsset
type BareMetalAssetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// How do we connect to the BMC?
	BMC BMCDetails `json:"bmc,omitempty"`

	// What is the name of the hardware profile for this host? It
	// should only be necessary to set this when inspection cannot
	// automatically determine the profile.
	HardwareProfile string `json:"hardwareProfile,omitempty"`

	// Which MAC address will PXE boot? This is optional for some
	// types, but required for libvirt VMs driven by vbmc.
	// +kubebuilder:validation:Pattern=`[0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){5}`
	BootMACAddress string `json:"bootMACAddress,omitempty"`

	// Role holds the role of the asset
	// +kubebuilder:validation:Enum=master;worker
	Role Role `json:"role,omitempty"`

	// ClusterDeployment which the asset belongs to.
	// +kubebuilder:pruning:PreserveUnknownFields
	ClusterDeployment metav1.ObjectMeta `json:"clusterDeployment,omitempty"`
}

// BareMetalAssetStatus defines the observed state of BareMetalAsset
type BareMetalAssetStatus struct {
	// Conditions describes the state of the BareMetalAsset resource.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`

	// RelatedObjects is a list of objects created and maintained by this
	// operator. Object references will be added to this list after they have
	// been created AND found in the cluster.
	// +optional
	RelatedObjects []corev1.ObjectReference `json:"relatedObjects,omitempty"`
}

// Condition Types
const (
	// ConditionCredentialsFound reports whether the secret containing the credentials
	// of a BareMetalAsset have been found.
	ConditionCredentialsFound string = "CredentialsFound"

	// ConditionAssetSyncStarted reports whether synchronization of a BareMetalHost
	// to a managed cluster has started
	ConditionAssetSyncStarted string = "AssetSyncStarted"

	// ConditionClusterDeploymentFound reports whether the cluster deployment referenced in
	// a BareMetalAsset has been found.
	ConditionClusterDeploymentFound string = "ClusterDeploymentFound"

	// ConditionAssetSyncCompleted reports whether synchronization of a BareMetalHost
	// to a managed cluster has completed
	ConditionAssetSyncCompleted string = "AssetSyncCompleted"
)

// Condition Reasons
const (
	ConditionReasonSecretNotFound            string = "SecretNotFound"
	ConditionReasonSecretFound               string = "SecretFound"
	ConditionReasonNoneSpecified             string = "NoneSpecified"
	ConditionReasonClusterDeploymentNotFound string = "ClusterDeploymentNotFound"
	ConditionReasonClusterDeploymentFound    string = "ClusterDeploymentFound"
	ConditionReasonSyncSetCreationFailed     string = "SyncSetCreationFailed"
	ConditionReasonSyncSetCreated            string = "SyncSetCreated"
	ConditionReasonSyncSetGetFailed          string = "SyncSetGetFailed"
	ConditionReasonSyncSetUpdateFailed       string = "SyncSetUpdateFailed"
	ConditionReasonSyncSetUpdated            string = "SyncSetUpdated"
	ConditionReasonSyncStatusNotFound        string = "SyncStatusNotFound"
	ConditionReasonSyncSetNotApplied         string = "SyncSetNotApplied"
	ConditionReasonSyncSetAppliedSuccessful  string = "SyncSetAppliedSuccessful"
	ConditionReasonSyncSetAppliedFailed      string = "SyncSetAppliedFailed"
	ConditionReasonUnexpectedResourceCount   string = "UnexpectedResourceCount"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BareMetalAsset is the Schema for the baremetalassets API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=baremetalassets,scope=Namespaced
type BareMetalAsset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BareMetalAssetSpec   `json:"spec,omitempty"`
	Status BareMetalAssetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BareMetalAssetList contains a list of BareMetalAsset
type BareMetalAssetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BareMetalAsset `json:"items"`
}
