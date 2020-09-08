package clustersetmapper

import (
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1alapha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	utils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/equals"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
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

var initMapperData = map[string][]string{
	"clusterSet1":  {"cluster11", "cluster12", "cluster13"},
	"clusterSet2":  {"cluster21", "cluster22", "cluster23"},
	"clusterSet12": {"cluster11", "cluster12", "cluster13", "cluster21", "cluster22", "cluster23"},
}

func newTestReconciler(managedClusterSetObjs, managedClusterObjs []runtime.Object) *Reconciler {
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
		clusterSetObjs     []runtime.Object
		clusterObjs        []runtime.Object
		expectedMapperData map[string][]string
		req                reconcile.Request
	}{
		{
			name: "update ClusterSet",
			clusterSetObjs: []runtime.Object{
				&clusterv1alapha1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterSet2",
					},
					Spec: clusterv1alapha1.ManagedClusterSetSpec{
						ClusterSelectors: []clusterv1alapha1.ClusterSelector{
							{
								ClusterNames: []string{"cluster31", "cluster11"},
							},
							{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"cloud": "aws",
									},
								},
							},
						},
					},
				},
			},
			clusterObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster31",
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster11",
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "cluster32",
						Labels: map[string]string{"cloud": "aws"},
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "cluster22",
						Labels: map[string]string{"cloud": "aws"},
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "clusterSet2",
				},
			},
			expectedMapperData: map[string][]string{
				"clusterSet1":  {"cluster11", "cluster12", "cluster13"},
				"clusterSet2":  {"cluster11", "cluster31", "cluster22", "cluster32"},
				"clusterSet12": {"cluster11", "cluster12", "cluster13", "cluster21", "cluster22", "cluster23"},
			},
		},
		{
			name:           "delete ClusterSet",
			clusterSetObjs: []runtime.Object{},
			clusterObjs:    []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "clusterSet2",
				},
			},
			expectedMapperData: map[string][]string{
				"clusterSet1":  {"cluster11", "cluster12", "cluster13"},
				"clusterSet12": {"cluster11", "cluster12", "cluster13", "cluster21", "cluster22", "cluster23"},
			},
		},
	}

	for _, test := range tests {
		r := newTestReconciler(test.clusterSetObjs, test.clusterObjs)
		r.Reconcile(test.req)
		validateResult(t, r, test.expectedMapperData)

	}
}

func validateResult(t *testing.T, r *Reconciler, expectedMapperData map[string][]string) {
	mapperData := r.clusterSetMapper.GetAllClusterSet2Clusters()
	assert.Equal(t, len(mapperData), len(expectedMapperData))
	for clusterSet, clusters := range mapperData {
		assert.True(t, utils.EqualStringSlice(clusters, expectedMapperData[clusterSet]))
	}
}
