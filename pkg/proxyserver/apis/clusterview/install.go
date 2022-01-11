package clusterview

import (
	clusterviewv1 "github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/clusterview/v1"
	clusterviewv1alpha1 "github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/clusterview/v1alpha1"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const GroupName = "clusterview.open-cluster-management.io"

var Scheme = runtime.NewScheme()

var Codecs = serializer.NewCodecFactory(Scheme)

var (
	// if you modify this, make sure you update the crEncoder
	unversionedVersion = schema.GroupVersion{Group: "", Version: "v1"}
	unversionedTypes   = []runtime.Object{
		&metav1.Status{},
		&metav1.WatchEvent{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
		&metav1.Table{},
	}
)

func init() {
	// we need to add the options to empty v1
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Group: "", Version: "v1"})
	metainternalversion.AddToScheme(Scheme)
	Scheme.AddUnversionedTypes(unversionedVersion, unversionedTypes...)

	Install(Scheme)
}

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func Install(scheme *runtime.Scheme) {
	utilruntime.Must(clusterviewv1.AddToScheme(scheme))
	utilruntime.Must(clusterviewv1alpha1.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(clusterviewv1.SchemeGroupVersion, clusterviewv1alpha1.SchemeGroupVersion))
}
