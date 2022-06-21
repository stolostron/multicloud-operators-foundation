package v1alpha1

const (
	// ClusterImageRegistryLabel value is namespace.managedClusterImageRegistry
	ClusterImageRegistryLabel = "open-cluster-management.io/image-registry"

	// ClusterImageRegistriesAnnotation value is a json string of ImageRegistries
	ClusterImageRegistriesAnnotation = "open-cluster-management.io/image-registries"
)

// ImageRegistries is value of the image registries annotation
type ImageRegistries struct {
	PullSecret string       `json:"pullSecret"`
	Registries []Registries `json:"registries"`
}
