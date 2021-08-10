package imageregistry

import (
	"context"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterapiv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	clusterv1alaph1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

var (
	scheme = runtime.NewScheme()
)

var (
	conditionPlacementGroupNotFound = metav1.Condition{
		Type:   v1alpha1.ConditionPlacementAvailable,
		Status: metav1.ConditionFalse,
		Reason: v1alpha1.ConditionReasonPlacementGroupNotFound,
	}
	conditionPlacementResourceNotFound = metav1.Condition{
		Type:   v1alpha1.ConditionPlacementAvailable,
		Status: metav1.ConditionFalse,
		Reason: v1alpha1.ConditionReasonPlacementResourceNotFound,
	}
	conditionSelectedTrue = metav1.Condition{
		Type:   v1alpha1.ConditionClustersSelected,
		Status: metav1.ConditionTrue,
		Reason: v1alpha1.ConditionReasonClusterSelected,
	}
	conditionUpdatedTrue = metav1.Condition{
		Type:   v1alpha1.ConditionClustersUpdated,
		Status: metav1.ConditionTrue,
		Reason: v1alpha1.ConditionReasonClustersUpdated,
	}
)

func init() {
	_ = clusterv1.Install(scheme)
	_ = clusterv1alaph1.Install(scheme)
	_ = v1alpha1.AddToScheme(scheme)
}

func newFakeReconciler(existingObjs []runtime.Object) *Reconciler {
	fakeClient := fake.NewClientBuilder()
	return &Reconciler{
		client: fakeClient.WithScheme(scheme).WithRuntimeObjects(existingObjs...).Build(),
		scheme: scheme,
	}
}

func newCluster(name, imageRegistry string) *clusterv1.ManagedCluster {
	cluster := &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if imageRegistry != "" {
		cluster.SetLabels(map[string]string{ClusterImageRegistryLabel: imageRegistry})
	}
	return cluster
}

func newPlacementDecision(namespace, name, placementName string, clusters []string) *clusterv1alaph1.PlacementDecision {
	placementDecision := &clusterv1alaph1.PlacementDecision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{placementLabel: placementName},
		},
		Status: clusterv1alaph1.PlacementDecisionStatus{
			Decisions: []clusterv1alaph1.ClusterDecision{},
		},
	}
	for _, cluster := range clusters {
		placementDecision.Status.Decisions = append(placementDecision.Status.Decisions,
			clusterv1alaph1.ClusterDecision{ClusterName: cluster})
	}
	return placementDecision
}

func newImageRegistry(namespace, name, placement string, conditions []metav1.Condition) *v1alpha1.ManagedClusterImageRegistry {
	return &v1alpha1.ManagedClusterImageRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ImageRegistrySpec{
			Registry:   "quay.io/abc/",
			PullSecret: corev1.LocalObjectReference{Name: "pullSecret"},
			PlacementRef: v1alpha1.PlacementRef{
				Group:    clusterapiv1alpha1.GroupName,
				Resource: "placements",
				Name:     placement,
			},
		},
		Status: v1alpha1.ImageRegistryStatus{Conditions: conditions},
	}
}
func TestReconcile(t *testing.T) {
	tests := []struct {
		name               string
		clusters           []*clusterv1.ManagedCluster
		placementDecisions []*clusterv1alaph1.PlacementDecision
		imageRegistries    []*v1alpha1.ManagedClusterImageRegistry
		req                reconcile.Request
		expectedConditions []metav1.Condition
		expectedClusters   []*clusterv1.ManagedCluster
	}{
		{
			name:     "add registry labels to clusters successfully",
			clusters: []*clusterv1.ManagedCluster{newCluster("c1", ""), newCluster("c2", "")},
			placementDecisions: []*clusterv1alaph1.PlacementDecision{
				newPlacementDecision("ns1", "p1-1", "p1", []string{"c1", "c2"})},
			imageRegistries: []*v1alpha1.ManagedClusterImageRegistry{
				newImageRegistry("ns1", "r1", "p1", []metav1.Condition{})},
			req:                reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "ns1", Name: "r1"}},
			expectedClusters:   []*clusterv1.ManagedCluster{newCluster("c1", "ns1.r1"), newCluster("c2", "ns1.r1")},
			expectedConditions: []metav1.Condition{conditionSelectedTrue, conditionUpdatedTrue},
		},
		{
			name:               "remove registry labels from clusters successfully",
			clusters:           []*clusterv1.ManagedCluster{newCluster("c1", "ns1.r1"), newCluster("c2", "ns1.r1")},
			placementDecisions: []*clusterv1alaph1.PlacementDecision{},
			imageRegistries:    []*v1alpha1.ManagedClusterImageRegistry{},
			req:                reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "ns1", Name: "r1"}},
			expectedClusters:   []*clusterv1.ManagedCluster{newCluster("c1", ""), newCluster("c2", "")},
			expectedConditions: []metav1.Condition{},
		},
		{
			name:     "update registry labels of clusters successfully",
			clusters: []*clusterv1.ManagedCluster{newCluster("c1", ""), newCluster("c2", "ns2.r2"), newCluster("c3", "ns3.r3")},
			placementDecisions: []*clusterv1alaph1.PlacementDecision{
				newPlacementDecision("ns1", "p1-1", "p1", []string{"c1", "c2", "c3"}),
				newPlacementDecision("ns2", "p2-1", "p2", []string{"c2"})},
			imageRegistries: []*v1alpha1.ManagedClusterImageRegistry{
				newImageRegistry("ns1", "r1", "p1", []metav1.Condition{}),
				newImageRegistry("ns2", "r2", "p2", []metav1.Condition{})},
			req:                reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "ns1", Name: "r1"}},
			expectedClusters:   []*clusterv1.ManagedCluster{newCluster("c1", "ns1.r1"), newCluster("c2", "ns2.r2"), newCluster("c3", "ns1.r1")},
			expectedConditions: []metav1.Condition{conditionSelectedTrue, conditionUpdatedTrue},
		},
		{
			name:               "invalid placement group in imageRegistry",
			clusters:           []*clusterv1.ManagedCluster{},
			placementDecisions: []*clusterv1alaph1.PlacementDecision{},
			imageRegistries: []*v1alpha1.ManagedClusterImageRegistry{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "r1",
						Namespace: "ns1",
					},
					Spec: v1alpha1.ImageRegistrySpec{
						Registry:   "quay.io/abc/",
						PullSecret: corev1.LocalObjectReference{Name: "pullSecret"},
						PlacementRef: v1alpha1.PlacementRef{
							Group:    "app.open-cluster-management.io",
							Resource: "placements",
							Name:     "p1",
						},
					},
				},
			},
			req:                reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "ns1", Name: "r1"}},
			expectedClusters:   []*clusterv1.ManagedCluster{},
			expectedConditions: []metav1.Condition{conditionPlacementGroupNotFound},
		},
		{
			name:               "invalid placement resource in imageRegistry",
			clusters:           []*clusterv1.ManagedCluster{},
			placementDecisions: []*clusterv1alaph1.PlacementDecision{},
			imageRegistries: []*v1alpha1.ManagedClusterImageRegistry{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "r1",
						Namespace: "ns1",
					},
					Spec: v1alpha1.ImageRegistrySpec{
						Registry:   "quay.io/abc/",
						PullSecret: corev1.LocalObjectReference{Name: "pullSecret"},
						PlacementRef: v1alpha1.PlacementRef{
							Group:    "cluster.open-cluster-management.io",
							Resource: "place",
							Name:     "p1",
						},
					},
				},
			},
			req:                reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "ns1", Name: "r1"}},
			expectedClusters:   []*clusterv1.ManagedCluster{},
			expectedConditions: []metav1.Condition{conditionPlacementResourceNotFound},
		},
	}

	for _, test := range tests {
		existingObjs := []runtime.Object{}
		for _, cluster := range test.clusters {
			existingObjs = append(existingObjs, cluster)
		}
		for _, placementDecision := range test.placementDecisions {
			existingObjs = append(existingObjs, placementDecision)
		}
		for _, registry := range test.imageRegistries {
			existingObjs = append(existingObjs, registry)
		}

		r := newFakeReconciler(existingObjs)
		_, err := r.Reconcile(context.TODO(), test.req)
		assert.NoError(t, err)
		validateClusters(t, r.client, test.expectedClusters)
		validateConditions(t, r.client, test.req.Namespace, test.req.Name, test.expectedConditions)
	}
}

func validateClusters(t *testing.T, client client.Client, expectedClusters []*clusterv1.ManagedCluster) {
	for _, cluster := range expectedClusters {
		expectedLabels := cluster.GetLabels()
		realCluster := &clusterv1.ManagedCluster{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: cluster.Name}, realCluster)
		assert.NoError(t, err)
		realLabels := realCluster.GetLabels()
		assert.Equal(t, expectedLabels[ClusterImageRegistryLabel], realLabels[ClusterImageRegistryLabel])
	}
}

func validateConditions(t *testing.T, client client.Client, namespace, imageRegistryName string, expectedConditions []metav1.Condition) {
	if len(expectedConditions) == 0 {
		return
	}

	imageRegistry := &v1alpha1.ManagedClusterImageRegistry{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: imageRegistryName}, imageRegistry)
	assert.NoError(t, err)
	assert.Equal(t, len(imageRegistry.Status.Conditions), len(expectedConditions))
	for _, condition := range expectedConditions {
		assert.True(t, meta.IsStatusConditionPresentAndEqual(imageRegistry.Status.Conditions, condition.Type, condition.Status))
	}
}
