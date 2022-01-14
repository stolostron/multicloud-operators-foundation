package clusterdeployment

import (
	"context"
	"os"
	"testing"

	utils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	clusterv1alapha1 "open-cluster-management.io/api/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
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
	if err := clusterv1.Install(scheme); err != nil {
		klog.Errorf("Failed adding cluster to scheme, %v", err)
		os.Exit(1)
	}
	if err := hivev1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding hive to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

func newTestReconciler(managedClusters, clusterpoolObjs, clusterdeploymentObjs []runtime.Object) *Reconciler {
	objs := append(clusterdeploymentObjs, clusterpoolObjs...)
	objs = append(objs, managedClusters...)
	r := &Reconciler{
		client: fake.NewFakeClientWithScheme(scheme, objs...),
		scheme: scheme,
	}
	return r
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name                  string
		clusterdeploymentObjs []runtime.Object
		clusterPools          []runtime.Object
		managedClusters       []runtime.Object
		expectedlabel         map[string]string
		req                   reconcile.Request
	}{
		{
			name: "add clusterdeployment",
			clusterdeploymentObjs: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "clusterdeployment1",
						Namespace: "ns1",
					},
					Spec: hivev1.ClusterDeploymentSpec{
						ClusterPoolRef: &hivev1.ClusterPoolReference{
							Namespace: "poolNs1",
							PoolName:  "pool1",
						},
					},
				},
			},
			clusterPools: []runtime.Object{
				&hivev1.ClusterPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pool1",
						Namespace: "poolNs1",
						Labels: map[string]string{
							utils.ClusterSetLabel: "clusterSet1",
						},
					},
					Spec: hivev1.ClusterPoolSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "clusterdeployment1",
					Namespace: "ns1",
				},
			},
			expectedlabel: map[string]string{
				utils.ClusterSetLabel: "clusterSet1",
			},
		},
		{
			name: "clusterdeployment related clusterpool has no set label",
			clusterdeploymentObjs: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "clusterdeployment1",
						Namespace: "ns1",
						Labels: map[string]string{
							utils.ClusterSetLabel: "clusterSet1",
						},
					},
					Spec: hivev1.ClusterDeploymentSpec{
						ClusterPoolRef: &hivev1.ClusterPoolReference{
							Namespace: "poolNs1",
							PoolName:  "pool1",
						},
					},
				},
			},
			clusterPools: []runtime.Object{
				&hivev1.ClusterPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pool1",
						Namespace: "poolNs1",
					},
					Spec: hivev1.ClusterPoolSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "clusterdeployment1",
					Namespace: "ns1",
				},
			},
			expectedlabel: map[string]string{},
		},
		{
			name: "clusterdeployment related clusterpool has been claimed, has managedcluster",
			clusterdeploymentObjs: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cd1",
						Namespace: "cd1",
					},
					Spec: hivev1.ClusterDeploymentSpec{
						ClusterPoolRef: &hivev1.ClusterPoolReference{
							Namespace: "poolNs1",
							PoolName:  "pool1",
							ClaimName: "claimed",
						},
					},
				},
			},
			clusterPools: []runtime.Object{
				&hivev1.ClusterPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pool1",
						Namespace: "poolNs1",
						Labels: map[string]string{
							utils.ClusterSetLabel: "clusterSet1",
						},
					},
					Spec: hivev1.ClusterPoolSpec{},
				},
			},
			managedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd1",
						Labels: map[string]string{
							utils.ClusterSetLabel: "clusterSet1",
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "cd1",
					Namespace: "cd1",
				},
			},
			expectedlabel: map[string]string{
				utils.ClusterSetLabel: "clusterSet1",
			},
		},
		{
			name: "directly create managedcluster, and clusterdeployment",
			clusterdeploymentObjs: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cd1",
						Namespace: "cd1",
					},
					Spec: hivev1.ClusterDeploymentSpec{},
				},
			},

			managedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd1",
						Labels: map[string]string{
							utils.ClusterSetLabel: "clusterSet1",
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "cd1",
					Namespace: "cd1",
				},
			},
			expectedlabel: map[string]string{
				utils.ClusterSetLabel: "clusterSet1",
			},
		},
	}

	for _, test := range tests {
		r := newTestReconciler(test.managedClusters, test.clusterPools, test.clusterdeploymentObjs)
		r.Reconcile(ctx, test.req)
		validateResult(t, r, test.name, test.req, test.expectedlabel)
	}
}

func validateResult(t *testing.T, r *Reconciler, caseName string, req reconcile.Request, expectedlabels map[string]string) {
	clusterdeployment := &hivev1.ClusterDeployment{}

	err := r.client.Get(context.TODO(), req.NamespacedName, clusterdeployment)
	if err != nil {
		t.Errorf("case: %v, failed to get clusterdeployment: %v", caseName, req.NamespacedName)
	}

	if len(clusterdeployment.Labels) != len(expectedlabels) {
		t.Errorf("case: %v, expect:%v  actual:%v", caseName, expectedlabels, clusterdeployment.Labels)
	}
}
