package clustersetmapper

import (
	"os"
	"testing"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1alapha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	scheme = runtime.NewScheme()
)

func TestMain(m *testing.M) {
	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, clusterv1.Install, clusterv1alapha1.Install)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := clusterv1alapha1.Install(scheme); err != nil {
		klog.Errorf("Failed adding cluster v1alph1 to scheme, %v", err)
		os.Exit(1)
	}
	if err := clusterv1.Install(scheme); err != nil {
		klog.Errorf("Failed adding cluster to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

func newTestReconciler(managedClusterSetObjs, managedClusterObjs []runtime.Object, initMapperData map[string]sets.String) *Reconciler {
	objs := managedClusterSetObjs
	objs = append(objs, managedClusterObjs...)
	r := &Reconciler{
		client:           fake.NewFakeClientWithScheme(scheme, objs...),
		scheme:           scheme,
		clusterSetMapper: helpers.NewClusterSetMapper(),
	}

	for clusterSet, clusters := range initMapperData {
		r.clusterSetMapper.UpdateClusterSetByClusters(clusterSet, clusters)
	}

	return r
}

func TestReconcile(t *testing.T) {

	tests := []struct {
		name               string
		initMap            map[string]sets.String
		clusterSetObjs     []runtime.Object
		clusterObjs        []runtime.Object
		expectedMapperData map[string]sets.String
		req                reconcile.Request
	}{
		{
			name: "add Cluster",
			initMap: map[string]sets.String{
				"clusterSet1": {"cluster2": {}, "cluster3": {}},
			},
			clusterSetObjs: []runtime.Object{
				&clusterv1alapha1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterSet1",
					},
					Spec: clusterv1alapha1.ManagedClusterSetSpec{},
				},
			},
			clusterObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster1",
						Labels: map[string]string{
							clusterSetLabel: "clusterSet1",
						},
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "cluster1",
				},
			},
			expectedMapperData: map[string]sets.String{
				"clusterSet1": {"cluster1": {}, "cluster2": {}, "cluster3": {}}},
		},
		{
			name: "update Cluster",
			initMap: map[string]sets.String{
				"clusterSet1": {"cluster2": {}, "cluster3": {}},
			},
			clusterSetObjs: []runtime.Object{
				&clusterv1alapha1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterSet2",
					},
					Spec: clusterv1alapha1.ManagedClusterSetSpec{},
				},
			},
			clusterObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster2",
						Labels: map[string]string{
							clusterSetLabel: "clusterSet2",
						},
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "cluster2",
				},
			},
			expectedMapperData: map[string]sets.String{
				"clusterSet1": {"cluster3": {}},
				"clusterSet2": {"cluster2": {}},
			},
		},
	}

	for _, test := range tests {
		r := newTestReconciler(test.clusterSetObjs, test.clusterObjs, test.initMap)
		r.Reconcile(test.req)
		validateResult(t, r, test.expectedMapperData)

	}
}

func validateResult(t *testing.T, r *Reconciler, expectedMapperData map[string]sets.String) {
	mapperData := r.clusterSetMapper.GetAllClusterSetToClusters()
	if !assert.Equal(t, len(mapperData), len(expectedMapperData)) {
		t.Errorf("expect:%v  actual:%v", expectedMapperData, mapperData)
	}
	for clusterSet, clusters := range mapperData {
		assert.True(t, expectedMapperData[clusterSet].Equal(clusters))
	}
}
