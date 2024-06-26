package constants

const (
	// ClusterClaimHostedClusterCountZero is the cluster claim key to represent whether there is no(zero)
	// hypershift hosted cluster hosted by the current cluster.
	//
	// hypershift-addon-agent will set this clusterclaim to the hosting managedcluster
	ClusterClaimHostedClusterCountZero = "zero.hostedclustercount.hypershift.openshift.io"

	// LabelFeatureHypershiftAddon is the feature for managed cluster to indicate whether the hypershift
	// addon is available on this managed cluster
	LabelFeatureHypershiftAddon = "feature.open-cluster-management.io/addon-hypershift-addon"

	// GlobalNamespaceAnnotation is the annotation on the global managed cluster set. If the cluster set has
	// this annotation, the related ns/binding/placement will not be created.
	GlobalNamespaceAnnotation = "open-cluster-management.io/ns-create"
)
