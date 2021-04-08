package clusterclaim

import (
	"os"
	"testing"

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

	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
)

var (
	scheme = runtime.NewScheme()
)

func TestMain(m *testing.M) {
	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}
	if err := clusterv1alapha1.Install(scheme); err != nil {
		klog.Errorf("Failed adding cluster v1alph1 to scheme, %v", err)
		os.Exit(1)
	}

	if err := hivev1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding hive to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

func newTestReconciler(managedClusterSetObjs, managedClusterObjs []runtime.Object, clusterpoolObjs []runtime.Object, initMapperData map[string]sets.String) *Reconciler {
	objs := managedClusterSetObjs
	objs = append(objs, managedClusterObjs...)
	objs = append(objs, clusterpoolObjs...)
	r := &Reconciler{
		client:           fake.NewFakeClientWithScheme(scheme, objs...),
		scheme:           scheme,
		clusterSetMapper: helpers.NewClusterSetMapper(),
	}

	for clusterSet, ns := range initMapperData {
		r.clusterSetMapper.UpdateClusterSetByObjects(clusterSet, ns)
	}

	return r
}

func TestReconcile(t *testing.T) {

	tests := []struct {
		name               string
		initMap            map[string]sets.String
		clusterSetObjs     []runtime.Object
		clusterClaimObjs   []runtime.Object
		clusterPools       []runtime.Object
		expectedMapperData map[string]sets.String
		req                reconcile.Request
	}{
		{
			name: "add Clusterclaim",
			initMap: map[string]sets.String{
				"clusterSet1": {
					"clusterclaims/ns1/clusterClaim2": {},
					"clusterclaims/ns2/clusterClaim3": {},
				},
			},
			clusterSetObjs: []runtime.Object{
				&clusterv1alapha1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterSet1",
					},
					Spec: clusterv1alapha1.ManagedClusterSetSpec{},
				},
			},
			clusterClaimObjs: []runtime.Object{
				&hivev1.ClusterClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "clusterClaim1",
						Namespace: "ns1",
						Labels: map[string]string{
							ClusterSetLabel: "clusterSet1",
						},
					},
					Spec: hivev1.ClusterClaimSpec{
						ClusterPoolName: "pool1",
					},
				},
			},
			clusterPools: []runtime.Object{
				&hivev1.ClusterPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pool1",
						Namespace: "ns1",
						Labels: map[string]string{
							ClusterSetLabel: "clusterSet1",
						},
					},
					Spec: hivev1.ClusterPoolSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "clusterClaim1",
					Namespace: "ns1",
				},
			},
			expectedMapperData: map[string]sets.String{
				"clusterSet1": {
					"clusterclaims/ns1/clusterClaim1": {},
					"clusterclaims/ns1/clusterClaim2": {},
					"clusterclaims/ns2/clusterClaim3": {},
				},
			},
		},
		{
			name: "update Clusterpool clusterset label",
			initMap: map[string]sets.String{
				"clusterSet1": {
					"clusterclaims/ns1/clusterClaim1": {},
					"clusterclaims/ns2/clusterClaim2": {},
				},
			},
			clusterSetObjs: []runtime.Object{
				&clusterv1alapha1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterSet2",
					},
					Spec: clusterv1alapha1.ManagedClusterSetSpec{},
				},
			},
			clusterClaimObjs: []runtime.Object{
				&hivev1.ClusterClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "clusterClaim1",
						Namespace: "ns1",
					},
					Spec: hivev1.ClusterClaimSpec{
						ClusterPoolName: "pool1",
					},
				},
			},
			clusterPools: []runtime.Object{
				&hivev1.ClusterPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pool1",
						Namespace: "ns1",
						Labels: map[string]string{
							ClusterSetLabel: "clusterSet2",
						},
					},
					Spec: hivev1.ClusterPoolSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "clusterClaim1",
					Namespace: "ns1",
				},
			},
			expectedMapperData: map[string]sets.String{
				"clusterSet1": {
					"clusterclaims/ns2/clusterClaim2": {},
				},
				"clusterSet2": {
					"clusterclaims/ns1/clusterClaim1": {},
				},
			},
		},
		{
			name: "delete Clusterpool clusterset label",
			initMap: map[string]sets.String{
				"clusterSet1": {
					"clusterclaims/ns1/clusterClaim1": {},
					"clusterclaims/ns2/clusterClaim2": {},
				},
			},
			clusterClaimObjs: []runtime.Object{
				&hivev1.ClusterClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "clusterClaim1",
						Namespace: "ns1",
					},
					Spec: hivev1.ClusterClaimSpec{
						ClusterPoolName: "pool1",
					},
				},
			},
			clusterPools: []runtime.Object{
				&hivev1.ClusterPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pool1",
						Namespace: "ns1",
					},
					Spec: hivev1.ClusterPoolSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "clusterClaim1",
					Namespace: "ns1",
				},
			},
			expectedMapperData: map[string]sets.String{
				"clusterSet1": {
					"clusterclaims/ns2/clusterClaim2": {},
				},
			},
		},
	}

	for _, test := range tests {
		r := newTestReconciler(test.clusterSetObjs, test.clusterClaimObjs, test.clusterPools, test.initMap)
		r.Reconcile(test.req)
		validateResult(t, r, test.name, test.expectedMapperData)

	}
}

func validateResult(t *testing.T, r *Reconciler, caseName string, expectedMapperData map[string]sets.String) {
	mapperData := r.clusterSetMapper.GetAllClusterSetToObjects()
	if !assert.Equal(t, len(mapperData), len(expectedMapperData)) {
		t.Errorf("case: %v, expect:%v  actual:%v", caseName, expectedMapperData, mapperData)
	}
	for clusterSet, clusters := range mapperData {
		if !assert.True(t, expectedMapperData[clusterSet].Equal(clusters)) {
			t.Errorf("case: %v, expect:%v  actual:%v", caseName, expectedMapperData, mapperData)
		}
	}
}
