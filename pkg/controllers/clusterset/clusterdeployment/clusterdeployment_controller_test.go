package clusterdeployment

import (
	"context"
	"os"
	"testing"

	utils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"

	clusterv1alapha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
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

	if err := hivev1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding hive to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

func newTestReconciler(clusterpoolObjs []runtime.Object, clusterdeploymentObjs []runtime.Object) *Reconciler {
	objs := append(clusterdeploymentObjs, clusterpoolObjs...)
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
			name: "clusterdeployment related clusterpool has been claimed",
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
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "clusterdeployment1",
					Namespace: "ns1",
				},
			},
			expectedlabel: map[string]string{},
		},
	}

	for _, test := range tests {
		r := newTestReconciler(test.clusterPools, test.clusterdeploymentObjs)
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
