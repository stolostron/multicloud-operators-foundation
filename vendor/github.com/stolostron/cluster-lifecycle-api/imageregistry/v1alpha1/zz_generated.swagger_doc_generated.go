package v1alpha1

// This file contains a collection of methods that can be used from go-restful to
// generate Swagger API documentation for its models. Please read this PR for more
// information on the implementation: https://github.com/emicklei/go-restful/pull/215
//
// TODOs are ignored from the parser (e.g. TODO(andronat):... || TODO:...) if and only if
// they are on one line! For multiple line or blocks that you want to ignore use ---.
// Any context after a --- is ignored.
//
// Those methods can be generated by using hack/update-swagger-docs.sh

// AUTO-GENERATED FUNCTIONS START HERE
var map_ImageRegistrySpec = map[string]string{
	"":             "ImageRegistrySpec is the spec of managedClusterImageRegistry.",
	"registry":     "Registry is the Mirror registry which will replace all images registries. will be ignored if Registries is not empty.",
	"registries":   "Registries includes the mirror and source registries. The source registry will be replaced by the Mirror. The larger index will work if the Sources are the same.",
	"pullSecret":   "PullSecret is the name of image pull secret which should be in the same namespace with the managedClusterImageRegistry.",
	"placementRef": "PlacementRef is the referred Placement name.",
}

func (ImageRegistrySpec) SwaggerDoc() map[string]string {
	return map_ImageRegistrySpec
}

var map_ImageRegistryStatus = map[string]string{
	"conditions": "Conditions contains condition information for a managedClusterImageRegistry",
}

func (ImageRegistryStatus) SwaggerDoc() map[string]string {
	return map_ImageRegistryStatus
}

var map_ManagedClusterImageRegistry = map[string]string{
	"":       "ManagedClusterImageRegistry represents the image overridden configuration information.",
	"spec":   "Spec defines the information of the ManagedClusterImageRegistry.",
	"status": "Status represents the desired status of the managedClusterImageRegistry.",
}

func (ManagedClusterImageRegistry) SwaggerDoc() map[string]string {
	return map_ManagedClusterImageRegistry
}

var map_ManagedClusterImageRegistryList = map[string]string{
	"":         "ManagedClusterImageRegistryList is a list of ManagedClusterImageRegistry objects.",
	"metadata": "Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
	"items":    "List of ManagedClusterInfo objects.",
}

func (ManagedClusterImageRegistryList) SwaggerDoc() map[string]string {
	return map_ManagedClusterImageRegistryList
}

var map_PlacementRef = map[string]string{
	"":         "PlacementRef is the referred placement",
	"group":    "Group is the api group of the placement. Current group is cluster.open-cluster-management.io.",
	"resource": "Resource is the resource type of the Placement. Current resource is placement or placements.",
	"name":     "Name is the name of the Placement.",
}

func (PlacementRef) SwaggerDoc() map[string]string {
	return map_PlacementRef
}

var map_Registries = map[string]string{
	"mirror": "Mirror is the mirrored registry of the Source. Will be ignored if Mirror is empty.",
	"source": "Source is the source registry. All image registries will be replaced by Mirror if Source is empty.",
}

func (Registries) SwaggerDoc() map[string]string {
	return map_Registries
}

// AUTO-GENERATED FUNCTIONS END HERE