package internalversion

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/printers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/diff"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"reflect"
	"testing"
	"time"
)

type TestPrintHandler struct {
	numCalls int
}

func (t *TestPrintHandler) TableHandler(columnDefinitions []metav1.TableColumnDefinition, printFunc interface{}) error {
	t.numCalls++
	return nil
}

func (t *TestPrintHandler) getNumCalls() int {
	return t.numCalls
}

func TestAllHandlers(t *testing.T) {
	h := &TestPrintHandler{numCalls: 0}
	AddHandlers(h)
	if h.getNumCalls() == 0 {
		t.Error("TableHandler not called in AddHandlers")
	}
}

func Test_printManagedCluster(t *testing.T) {
	tests := []struct {
		cluster  clusterv1.ManagedCluster
		options  printers.GenerateOptions
		expected []metav1.TableRow
	}{
		{
			cluster: clusterv1.ManagedCluster{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "cluster1",
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-3e11)},
				},
				Spec: clusterv1.ManagedClusterSpec{
					ManagedClusterClientConfigs: []clusterv1.ClientConfig{
						{
							URL:      "http://localhost",
							CABundle: nil,
						},
					},
					HubAcceptsClient:     true,
					LeaseDurationSeconds: 60,
				},
				Status: clusterv1.ManagedClusterStatus{
					Conditions: []metav1.Condition{
						{
							Type:   clusterv1.ManagedClusterConditionJoined,
							Status: "True",
						},
						{
							Type:   clusterv1.ManagedClusterConditionAvailable,
							Status: "False",
						},
					},
				},
			},
			expected: []metav1.TableRow{
				{Cells: []interface{}{"cluster1", true, "http://localhost", "True", "False", "5m"}},
			},
		},
	}

	for i, test := range tests {
		rows, err := printManagedCluster(&test.cluster, test.options)
		if err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			rows[i].Object.Object = nil
		}
		if !reflect.DeepEqual(test.expected, rows) {
			t.Errorf("%d mismatch: %s", i, diff.ObjectReflectDiff(test.expected, rows))
		}
	}
}

func Test_printManagedClusterList(t *testing.T) {
	tests := []struct {
		clusterList clusterv1.ManagedClusterList
		options     printers.GenerateOptions
		expected    []metav1.TableRow
	}{
		{
			clusterList: clusterv1.ManagedClusterList{

				Items: []clusterv1.ManagedCluster{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:              "cluster1",
							CreationTimestamp: metav1.Time{Time: time.Now().Add(-3e11)},
						},
						Spec: clusterv1.ManagedClusterSpec{
							ManagedClusterClientConfigs: []clusterv1.ClientConfig{
								{
									URL:      "http://localhost",
									CABundle: nil,
								},
							},
							HubAcceptsClient:     true,
							LeaseDurationSeconds: 60,
						},
						Status: clusterv1.ManagedClusterStatus{
							Conditions: []metav1.Condition{
								{
									Type:   clusterv1.ManagedClusterConditionJoined,
									Status: "True",
								},
								{
									Type:   clusterv1.ManagedClusterConditionAvailable,
									Status: "False",
								},
							},
						},
					},
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:              "cluster2",
							CreationTimestamp: metav1.Time{Time: time.Now().Add(-3e11)},
						},
						Spec: clusterv1.ManagedClusterSpec{
							ManagedClusterClientConfigs: []clusterv1.ClientConfig{
								{
									URL:      "http://localhost",
									CABundle: nil,
								},
							},
							HubAcceptsClient:     true,
							LeaseDurationSeconds: 60,
						},
						Status: clusterv1.ManagedClusterStatus{
							Conditions: []metav1.Condition{
								{
									Type:   clusterv1.ManagedClusterConditionJoined,
									Status: "True",
								},
								{
									Type:   clusterv1.ManagedClusterConditionAvailable,
									Status: "True",
								},
							},
						},
					},
				},
			},

			expected: []metav1.TableRow{
				{Cells: []interface{}{"cluster1", true, "http://localhost", "True", "False", "5m"}},
				{Cells: []interface{}{"cluster2", true, "http://localhost", "True", "True", "5m"}},
			},
		},
	}

	for i, test := range tests {
		rows, err := printManagedClusterList(&test.clusterList, test.options)
		if err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			rows[i].Object.Object = nil
		}
		if !reflect.DeepEqual(test.expected, rows) {
			t.Errorf("%d mismatch: %s", i, diff.ObjectReflectDiff(test.expected, rows))
		}
	}
}
