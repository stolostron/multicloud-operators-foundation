package constants

const (
	// AnnotationKlusterletDeployMode is the annotation key of klusterlet deploy mode, it describes the
	// klusterlet deploy mode when importing a managed cluster.
	// If the value is "Hosted", the HostingClusterNameAnnotation annotation will be required, we use
	// AnnotationKlusterletHostingClusterName to determine where to deploy the registration-agent and
	// work-agent.
	AnnotationKlusterletDeployMode string = "import.open-cluster-management.io/klusterlet-deploy-mode"

	// AnnotationKlusterletHostingClusterName is the annotation key of hosting cluster name for klusterlet,
	// it is required in Hosted mode, and the hosting cluster MUST be one of the managed cluster of the hub.
	// The value of the annotation should be the ManagedCluster name of the hosting cluster.
	AnnotationKlusterletHostingClusterName string = "import.open-cluster-management.io/hosting-cluster-name"

	// DisableAutoImportAnnotation is an annotation of ManagedCluster.
	// If present, the crds.yaml and import.yaml will not be applied on the managed cluster by the hub
	// controller automatically. And the bootstrap-hub-kubeconfig secret will not be updated as well
	// in the backup-restore case.
	DisableAutoImportAnnotation string = "import.open-cluster-management.io/disable-auto-import"

	// AnnotationKlusterletConfig is an annotation of ManagedCluster, which references to the name of the
	// KlusterletConfig adopted by this managed cluster. If it is missing on a ManagedCluster, no KlusterletConfig
	// will be used for this managed cluster.
	AnnotationKlusterletConfig string = "agent.open-cluster-management.io/klusterlet-config"

	// SelfManagedClusterLabelKey is the label key on the ManagedCluster resource to tag it as the local cluster managed
	// by the ACM hub. Only one ManagedCluster and only the ACM hub cluster can have this label.
	SelfManagedClusterLabelKey string = "local-cluster"

	// HubCABundleLabelKey is the label key on configmaps to store the customized ca bundle.
	HubCABundleLabelKey string = "import.open-cluster-management.io/ca-bundle"

	// AutoImportStrategyImportAndSync is an AutoImportStrategy. If the cluster has not yet joined the
	// hub, the hub controller imports it by applying klusterlet manifests on the managed cluster. Once
	// joined, both the hub controller and the klusterlet agent will continue to synchronize the klusterlet
	// manifests on the managed cluster with the hub configuration.
	AutoImportStrategyImportAndSync = "ImportAndSync"

	// AutoImportStrategyImportOnly is an AutoImportStrategy. If the cluster has not yet joined the hub,
	// the hub controller imports it by applying klusterlet manifests on the managed cluster. After
	// the cluster has joined, the hub controller will no longer apply klusterlet manifests, and only
	// the klusterlet agent will sync the hub configuration to the managed cluster.
	AutoImportStrategyImportOnly = "ImportOnly"

	// AnnotationImmediateImport is an annotation for ManagedCluster. When added with an empty value,
	// it immediately triggers an import process regardless of the AutoImportStrategy setting or the
	// current import state of the ManagedCluster. However, this has no effect if the
	// ManagedCluster resource contains the `import.open-cluster-management.io/disable-auto-import` annotation.
	// Upon successful import, the hub controller updates the annotation value to `Completed`.
	AnnotationImmediateImport = "import.open-cluster-management.io/immediate-import"

	// AnnotationValueImmediateImportCompleted is the value set by the hub controller once the import process
	// trigged by annotation `import.open-cluster-management.io/immediate-import` is done.
	AnnotationValueImmediateImportCompleted = "Completed"
)
