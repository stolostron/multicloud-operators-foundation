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
)
