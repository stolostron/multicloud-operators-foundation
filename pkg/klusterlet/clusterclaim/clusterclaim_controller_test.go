package clusterclaim

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	clienttesting "k8s.io/client-go/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterv1alpha1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1alpha1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

type fakeClusterClaimLister struct {
	clusterClaims []*clusterv1alpha1.ClusterClaim
}

func (f *fakeClusterClaimLister) Get(name string) (*clusterv1alpha1.ClusterClaim, error) {
	for _, c := range f.clusterClaims {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, errors.NewNotFound(clusterv1alpha1.Resource("clusterclaim"), name)
}

func (f *fakeClusterClaimLister) List(selector labels.Selector) (ret []*clusterv1alpha1.ClusterClaim, err error) {
	for _, c := range f.clusterClaims {
		if selector.Matches(labels.Set(c.Labels)) {
			ret = append(ret, c)
		}
	}
	return
}

func newFakeClusterClaimLister(clusterClaims []*clusterv1alpha1.ClusterClaim) clusterv1alpha1lister.ClusterClaimLister {
	return &fakeClusterClaimLister{
		clusterClaims: clusterClaims,
	}
}

func TestCreateOrUpdate(t *testing.T) {
	testcases := []struct {
		name                 string
		objects              []runtime.Object
		clusterclaims        []*clusterv1alpha1.ClusterClaim
		validateAddonActions func(t *testing.T, actions []clienttesting.Action)
	}{
		{
			name:    "create cluster claim",
			objects: []runtime.Object{},
			clusterclaims: []*clusterv1alpha1.ClusterClaim{
				newClusterClaim("x", "y"),
			},
			validateAddonActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 2 {
					t.Errorf("Expect %d actions, but got: %v", 2, len(actions))
				}
				if actions[1].GetVerb() != "create" {
					t.Errorf("Expect action create, but got: %s", actions[1].GetVerb())
				}
			},
		},
		{
			name: "update cluster claim",
			objects: []runtime.Object{
				newClusterClaim("x", "y"),
			},
			clusterclaims: []*clusterv1alpha1.ClusterClaim{
				newClusterClaim("x", "z"),
			},
			validateAddonActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 2 {
					t.Errorf("Expect 2 actions, but got: %v", len(actions))
				}
				if actions[1].GetVerb() != "update" {
					t.Errorf("Expect action update, but got: %s", actions[1].GetVerb())
				}
			},
		},
		{
			name:    "update cluster claim with create only list with empty",
			objects: []runtime.Object{},
			clusterclaims: []*clusterv1alpha1.ClusterClaim{
				newClusterClaim(ClaimK8sID, "y"),
				newClusterClaim(ClaimK8sID, "z"),
			},
			validateAddonActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 3 {
					t.Errorf("Expect 3 actions, but got %d actions: %v", len(actions), actions)
				}
				if actions[0].GetVerb() != "get" {
					t.Errorf("Expect action get, but got: %s", actions[1].GetVerb())
				}
				if actions[1].GetVerb() != "create" {
					t.Errorf("Expect action create, but got: %s", actions[1].GetVerb())
				}
				if actions[2].GetVerb() != "get" {
					t.Errorf("Expect action get, but got: %s", actions[1].GetVerb())
				}
			},
		},
		{
			name: "update cluster claim with create only list",
			objects: []runtime.Object{
				newClusterClaim(ClaimK8sID, "y"),
			},
			clusterclaims: []*clusterv1alpha1.ClusterClaim{
				newClusterClaim(ClaimK8sID, "yy"),
			},
			validateAddonActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 1 {
					t.Errorf("Expect 1 actions, but got %d actions: %v", len(actions), actions)
				}
				if actions[0].GetVerb() != "get" {
					t.Errorf("Expect action get, but got: %s", actions[1].GetVerb())
				}
			},
		},
		{
			name: "update with the product and platform turns from a specific value to 'Other'",
			objects: []runtime.Object{
				newClusterClaim(ClaimOCMProduct, ProductROSA),
				newClusterClaim(ClaimOCMPlatform, PlatformAWS),
			},
			clusterclaims: []*clusterv1alpha1.ClusterClaim{
				newClusterClaim(ClaimOCMProduct, ProductOther),
				newClusterClaim(ClaimOCMPlatform, PlatformOther),
			},
			validateAddonActions: func(t *testing.T, actions []clienttesting.Action) {
				// expect 2 'get' actions
				if len(actions) != 2 {
					t.Errorf("Expect 2 actions, but got %d actions: %v", len(actions), actions)
				}
				if actions[0].GetVerb() != "get" {
					t.Errorf("Expect action get, but got: %s", actions[1].GetVerb())
				}
				if actions[1].GetVerb() != "get" {
					t.Errorf("Expect action get, but got: %s", actions[1].GetVerb())
				}
			},
		},
		{
			name: "update with the product and platform turns from 'Other' to another value",
			objects: []runtime.Object{
				newClusterClaim(ClaimOCMProduct, ProductOther),
				newClusterClaim(ClaimOCMPlatform, PlatformOther),
			},
			clusterclaims: []*clusterv1alpha1.ClusterClaim{
				newClusterClaim(ClaimOCMProduct, ProductROSA),
				newClusterClaim(ClaimOCMPlatform, PlatformAWS),
			},
			validateAddonActions: func(t *testing.T, actions []clienttesting.Action) {
				// expect 4 actions: get update get update
				if len(actions) != 4 {
					t.Errorf("Expect 3 actions, but got %d actions: %v", len(actions), actions)
				}
				if actions[0].GetVerb() != "get" {
					t.Errorf("Expect action get, but got: %s", actions[1].GetVerb())
				}
				if actions[1].GetVerb() != "update" {
					t.Errorf("Expect action update, but got: %s", actions[1].GetVerb())
				}
				if actions[2].GetVerb() != "get" {
					t.Errorf("Expect action get, but got: %s", actions[1].GetVerb())
				}
				if actions[3].GetVerb() != "update" {
					t.Errorf("Expect action update, but got: %s", actions[1].GetVerb())
				}
			},
		},
	}

	ctx := context.Background()
	for _, tc := range testcases {
		clusterClient := clusterfake.NewSimpleClientset(tc.objects...)
		for _, cc := range tc.clusterclaims {
			if err := createOrUpdateClusterClaim(ctx, clusterClient, cc, updateChecks); err != nil {
				t.Errorf("%s: unexpected error: %v", tc.name, err)
			}
		}
		tc.validateAddonActions(t, clusterClient.Actions())
	}
}

var testClusterName = "test-cluster"

func TestSyncControlPlaneCreatedClaims(t *testing.T) {
	ctx := context.Background()
	expected := []*clusterv1alpha1.ClusterClaim{
		newClusterClaim("x", "1"),
		newClusterClaim("y", "2"),
		newClusterClaim("z", "3"),
	}
	current := []*clusterv1alpha1.ClusterClaim{
		// Expect to be deleted after reconcile
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "hubManaged",
				Labels: map[string]string{labelHubManaged: ""},
			},
		},
		// Expect to be kept after reconcile
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "syncedLabel",
				Labels: map[string]string{labelHubManaged: "", labelCustomizedOnly: ""},
			},
		},
	}

	clusterClient := clusterfake.NewSimpleClientset(current[0], current[1])
	reconciler, _ := NewClusterClaimReconciler(
		ctrl.Log,
		testClusterName,
		clusterClient,
		fake.NewClientBuilder().Build(),
		newFakeClusterClaimLister(current),
		mockNodeLister{},
		func() ([]*clusterv1alpha1.ClusterClaim, error) {
			return expected, nil
		}, false)

	if err := reconciler.syncControlPlaneCreatedClaims(ctx); err != nil {
		t.Errorf("Failed to sync cluster claims: %v", err)
	}

	// Expect claims in 'expected' list are created or updated
	for _, item := range expected {
		claim, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.Background(), item.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Unable to find cluster claims: %s", item.Name)
		}

		if !reflect.DeepEqual(item.Spec, claim.Spec) {
			t.Errorf("Expected cluster claim %v, but got %v", item, claim)
		}
	}

	// Expect 'hubManaged' are deleted
	if _, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.Background(),
		"hubManaged", metav1.GetOptions{}); !errors.IsNotFound(err) {
		t.Errorf("deleted cluster claim hubManaged is not deleted")
	}

	// Expect 'syncedLabel' are not deleted
	if _, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.Background(),
		"syncedLabel", metav1.GetOptions{}); err != nil {
		t.Errorf("get cluster claim syncedLabel err: %v", err)
	}
}

func TestGenLabelsToClaims(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		claims []*clusterv1alpha1.ClusterClaim
	}{
		{
			name: "no label",
		},
		{
			name: "internal labels",
			labels: map[string]string{
				clusterv1beta1.LabelCloudVendor:     "a",
				clusterv1beta1.LabelKubeVendor:      "b",
				clusterv1beta1.LabelManagedBy:       "c",
				clusterv1beta1.OCPVersion:           "d",
				clusterv1beta1.OCPVersionMajor:      "4.x",
				clusterv1beta1.OCPVersionMajorMinor: "4.x",
			},
			claims: []*clusterv1alpha1.ClusterClaim{
				// ocpversionmajor and ocpversionmajorminor has been dependied by GRC, SD, and some customer TAMs. Should not be added into internalLabels set.
				newClusterClaim(strings.ToLower(clusterv1beta1.OCPVersionMajor), "4.x"),
				newClusterClaim(strings.ToLower(clusterv1beta1.OCPVersionMajorMinor), "4.x"),
			},
		},
		{
			name: "too long & empty value",
			labels: map[string]string{
				fmt.Sprintf("%s.com/xyz", strings.Repeat("x", 253)): "value",
				"empty-label": "",
			},
		},
		{
			name: "with transformation",
			labels: map[string]string{
				"UPPERCASE":   "UPPERCASE",
				"abc.com/def": "abc.com/def",
				"abc/def/ghi": "abc/def/ghi",
				"under_score": "under_score",
			},
			claims: []*clusterv1alpha1.ClusterClaim{
				newClusterClaim("uppercase", "UPPERCASE"),
				newClusterClaim("def.abc.com", "abc.com/def"),
				newClusterClaim("abc.def.ghi", "abc/def/ghi"),
				newClusterClaim("under-score", "under_score"),
			},
		},
		{
			name: "without transformation",
			labels: map[string]string{
				"label": "value",
			},
			claims: []*clusterv1alpha1.ClusterClaim{
				newClusterClaim("label", "value"),
			},
		},
		{
			name: "contains 'open-cluster-management.io'",
			labels: map[string]string{
				"open-cluster-management.io/agent": "klusterlet",
			},
			claims: []*clusterv1alpha1.ClusterClaim{},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(clusterv1beta1.SchemeGroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	_ = clusterv1beta1.AddToScheme(s)

	clusterName := "test-cluster"
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hubClient := fake.NewClientBuilder().WithScheme(s).
				WithRuntimeObjects(newFakeManagedClusterInfo(clusterName, test.labels)).Build()
			actual, err := genLabelsToClaims(hubClient, clusterName)
			assert.Equal(t, nil, err)
			assert.Equal(t, len(test.claims), len(actual))
			sort.SliceStable(test.claims, func(i, j int) bool { return test.claims[i].Name < test.claims[j].Name })
			sort.SliceStable(actual, func(i, j int) bool { return actual[i].Name < actual[j].Name })
			for i, claim := range test.claims {
				assert.Equal(t, claim.Name, actual[i].Name)
				assert.Equal(t, claim.Spec.Value, actual[i].Spec.Value)
			}
		})
	}
}

func newFakeManagedClusterInfo(clusterName string, labels map[string]string) *clusterv1beta1.ManagedClusterInfo {
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: clusterName,
			Labels: map[string]string{
				clusterv1beta1.LabelClusterID:   "1234",
				clusterv1beta1.LabelCloudVendor: "AWS",
				clusterv1beta1.LabelKubeVendor:  "OCP",
			},
		},
	}
	clusterInfo.SetLabels(labels)

	return clusterInfo
}

func TestSyncLabelsToClaims(t *testing.T) {
	var err error
	ctx := context.Background()
	current := []*clusterv1alpha1.ClusterClaim{
		// Expect to be kept after reconcile
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "hubManaged",
				Labels: map[string]string{labelHubManaged: ""},
			},
		},
		// Expect to be deleted after reconcile
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "syncedLabel",
				Labels: map[string]string{labelHubManaged: "", labelCustomizedOnly: ""},
			},
		},
	}

	// Reconciler will get expected(wating for sync) claims from managed cluster info labels
	managedClusterInfo := newFakeManagedClusterInfo(testClusterName, map[string]string{
		"os": "linux", // expect to be created after the reconcile
	})

	s := scheme.Scheme
	s.AddKnownTypes(clusterv1beta1.SchemeGroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	_ = clusterv1beta1.AddToScheme(s)

	// populate with the current claims
	hubClient := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(managedClusterInfo).Build()
	clusterClient := clusterfake.NewSimpleClientset(current[0], current[1])

	reconciler, err := NewClusterClaimReconciler(
		ctrl.Log,
		testClusterName,
		clusterClient,
		hubClient,
		newFakeClusterClaimLister(current),
		mockNodeLister{},
		func() ([]*clusterv1alpha1.ClusterClaim, error) {
			return current, nil
		}, true)

	if err != nil {
		t.Errorf("Failed to create cluster claim reconciler: %v", err)
		return
	}

	_, err = reconciler.Reconcile(ctx, ctrl.Request{})
	if err != nil {
		t.Errorf("Failed to sync cluster claims: %v", err)
		return
	}

	// hubManaged should not be ignored
	if _, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.Background(),
		"hubManaged", metav1.GetOptions{}); err != nil {
		t.Errorf("get cluster claim hubManaged err: %v", err)
	}

	// syncedLabel should be deleted
	if _, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.Background(),
		"syncedLabel", metav1.GetOptions{}); !errors.IsNotFound(err) {
		t.Errorf("deleted cluster claim syncedLabel is not deleted")
	}

	// os should be created
	if _, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.Background(),
		"os", metav1.GetOptions{}); err != nil {
		t.Errorf("get cluster claim os err: %v", err)
	}
}

func TestGetClusterSchedulable(t *testing.T) {
	testCases := []struct {
		name          string
		nodes         []*corev1.Node
		expected      bool
		expectedError error
	}{
		{
			name:          "No nodes",
			nodes:         []*corev1.Node{},
			expected:      false,
			expectedError: nil,
		},
		{
			name: "All nodes unschedulable",
			nodes: []*corev1.Node{
				{Spec: corev1.NodeSpec{Unschedulable: true}},
				{Spec: corev1.NodeSpec{Unschedulable: true}},
			},
			expected:      false,
			expectedError: nil,
		},
		{
			name: "Some nodes schedulable",
			nodes: []*corev1.Node{
				{Spec: corev1.NodeSpec{Unschedulable: true}},
				{Spec: corev1.NodeSpec{Unschedulable: false},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.NodeNetworkUnavailable,
								Status: corev1.ConditionFalse,
							},
						},
					}},
			},
			expected:      true,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodeLister := mockNodeLister{nodes: tc.nodes}
			result, err := getClusterSchedulable(nodeLister)
			if !reflect.DeepEqual(err, tc.expectedError) {
				t.Errorf("Expected error %v, but got %v", tc.expectedError, err)
			}
			if result != tc.expected {
				t.Errorf("Expected result %v, but got %v", tc.expected, result)
			}
		})
	}
}

type mockNodeLister struct {
	nodes []*corev1.Node
	err   error
}

func (m mockNodeLister) List(selector labels.Selector) ([]*corev1.Node, error) {
	return m.nodes, m.err
}

func (m mockNodeLister) Get(name string) (*corev1.Node, error) {
	return nil, nil
}
