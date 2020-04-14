package etcd

import (
	"testing"

	install "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/clusterregistry/install"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericregistrytest "k8s.io/apiserver/pkg/registry/generic/testing"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime/serializer"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"

	etcd3testing "k8s.io/apiserver/pkg/storage/etcd3/testing"
)

func newStorage(t *testing.T) (*REST, *etcd3testing.EtcdTestServer, *StatusREST) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	install.Install(scheme)
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1alpha1"})
	unversioned := schema.GroupVersion{Group: "clusterregistry.k8s.io", Version: "v1alpha1"}
	scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
		&metav1.ExportOptions{},
		&metav1.WatchEvent{},
		&metav1beta1.Table{})
	server, etcdStorage := etcd3testing.NewUnsecuredEtcd3TestClientServer(t)

	etcdStorage.Codec = codecs.LegacyCodec(
		schema.GroupVersion{Group: "clusterregistry.k8s.io", Version: "v1alpha1"})

	restOptions := generic.RESTOptions{
		StorageConfig:           etcdStorage,
		Decorator:               generic.UndecoratedStorage,
		DeleteCollectionWorkers: 1,
		ResourcePrefix:          "clusters",
	}
	rest, resstatus := NewREST(restOptions)
	return rest, server, resstatus
}

func validNewCluster() *clusterregistry.Cluster {
	return &clusterregistry.Cluster{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name: "cluster1",
		},
	}
}

func TestRest(t *testing.T) {
	storage, server, resstatus := newStorage(t)
	defer server.Terminate(t)
	defer storage.Store.DestroyFunc()
	ctx := genericapirequest.NewDefaultContext()
	test := genericregistrytest.New(t, storage.Store)
	test.TestCreate(
		validNewCluster(),
		&clusterregistry.Cluster{
			TypeMeta: v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{
				Name: "",
			},
		},
	)
	resstatus.New()
	resstatus.Get(ctx, "cluster1", &metav1.GetOptions{})
}
