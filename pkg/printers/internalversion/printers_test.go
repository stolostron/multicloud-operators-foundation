// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package internalversion

import (
	"reflect"
	"testing"
	"time"

	certificates "k8s.io/api/certificates/v1beta1"

	mcm "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	mcmv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/printers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistryv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/utils/diff"
)

type TestPrintHandler struct {
	numCalls int
}

var printOptions = printers.PrintOptions{}

func (t *TestPrintHandler) TableHandler(columnDefinitions []metav1.TableColumnDefinition, printFunc interface{}) error {
	t.numCalls++
	return nil
}
func (t *TestPrintHandler) DefaultTableHandler(columnDefinitions []metav1.TableColumnDefinition, printFunc interface{}) error {
	t.numCalls++
	return nil
}
func (t *TestPrintHandler) Handler(columns, columnsWithWide []string, printFunc interface{}) error {
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

func TestPrintCluster(t *testing.T) {
	tests := []struct {
		cluster  clusterregistryv1alpha1.Cluster
		expected []metav1.TableRow
	}{
		// Basic cluster
		{
			cluster: clusterregistryv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "cluster1",
					Namespace:         "cn1",
					CreationTimestamp: metav1.Time{Time: time.Now().Add(1.9e9)},
				},
				Spec: clusterregistryv1alpha1.ClusterSpec{
					KubernetesAPIEndpoints: clusterregistryv1alpha1.KubernetesAPIEndpoints{
						ServerEndpoints: []clusterregistryv1alpha1.ServerAddressByClientCIDR{
							{
								ServerAddress: "127.0.0.1:8001",
							},
						},
					},
				},
				Status: clusterregistryv1alpha1.ClusterStatus{
					Conditions: []clusterregistryv1alpha1.ClusterCondition{
						{},
					},
				},
			},

			expected: []metav1.TableRow{{Cells: []interface{}{"cluster1", "127.0.0.1:8001", "Offline", "0s"}}},
		},
		// Basic cluster without status or age.
		{
			cluster: clusterregistryv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "cluster2",
					Namespace:         "cn2",
					CreationTimestamp: metav1.Time{Time: time.Now().Add(1.9e9)},
				},
				Spec: clusterregistryv1alpha1.ClusterSpec{
					KubernetesAPIEndpoints: clusterregistryv1alpha1.KubernetesAPIEndpoints{
						ServerEndpoints: []clusterregistryv1alpha1.ServerAddressByClientCIDR{
							{
								ServerAddress: "127.0.0.1:8001",
							},
						},
					},
				},
				Status: clusterregistryv1alpha1.ClusterStatus{
					Conditions: []clusterregistryv1alpha1.ClusterCondition{
						{
							Type:   clusterregistryv1alpha1.ClusterOK,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},

			expected: []metav1.TableRow{{Cells: []interface{}{"cluster2", "127.0.0.1:8001", "Ready", "0s"}}},
		},
	}

	for i, test := range tests {
		rows, err := printCluster(&test.cluster, printOptions)
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

func TestPrintClusterList(t *testing.T) {
	tests := []struct {
		clusterList clusterregistryv1alpha1.ClusterList
		expected    []metav1.TableRow
	}{
		{
			clusterregistryv1alpha1.ClusterList{
				Items: []clusterregistryv1alpha1.Cluster{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "cluster1",
							Namespace:         "cn1",
							CreationTimestamp: metav1.Time{Time: time.Now().Add(1.9e9)},
						},
						Spec: clusterregistryv1alpha1.ClusterSpec{
							KubernetesAPIEndpoints: clusterregistryv1alpha1.KubernetesAPIEndpoints{
								ServerEndpoints: []clusterregistryv1alpha1.ServerAddressByClientCIDR{
									{
										ServerAddress: "127.0.0.1:8001",
									},
								},
							},
						},
						Status: clusterregistryv1alpha1.ClusterStatus{
							Conditions: []clusterregistryv1alpha1.ClusterCondition{},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "cluster2",
							Namespace:         "cn2",
							CreationTimestamp: metav1.Time{Time: time.Now().Add(1.9e9)},
						},
						Spec: clusterregistryv1alpha1.ClusterSpec{
							KubernetesAPIEndpoints: clusterregistryv1alpha1.KubernetesAPIEndpoints{
								ServerEndpoints: []clusterregistryv1alpha1.ServerAddressByClientCIDR{
									{
										ServerAddress: "127.0.0.1:8001",
									},
								},
							},
						},
						Status: clusterregistryv1alpha1.ClusterStatus{
							Conditions: []clusterregistryv1alpha1.ClusterCondition{
								{
									Type:   clusterregistryv1alpha1.ClusterOK,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			[]metav1.TableRow{
				{
					Cells: []interface{}{"cluster1", "127.0.0.1:8001", "Pending", "0s"},
				},
				{
					Cells: []interface{}{"cluster2", "127.0.0.1:8001", "Ready", "0s"},
				},
			},
		},
	}

	for _, test := range tests {
		rows, err := printClusterList(&test.clusterList, printOptions)

		if err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			rows[i].Object.Object = nil
		}
		if !reflect.DeepEqual(test.expected, rows) {
			t.Errorf("mismatch: %s", diff.ObjectReflectDiff(test.expected, rows))
		}
	}
}

func TestPrintClusterStatus(t *testing.T) {
	tests := []struct {
		clusterStatus mcm.ClusterStatus
		expected      []metav1.TableRow
	}{
		// Basic cluster
		{
			clusterStatus: mcm.ClusterStatus{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "clusterstatus1",
					Namespace:         "cn1",
					CreationTimestamp: metav1.Time{Time: time.Now().Add(1.9e9)},
				},
				Spec: mcm.ClusterStatusSpec{
					Capacity: corev1.ResourceList{
						corev1.ResourceCPU:        *resource.NewQuantity(int64(16), resource.BinarySI),
						corev1.ResourceMemory:     *resource.NewQuantity(int64(1024), resource.BinarySI),
						corev1.ResourceStorage:    *resource.NewQuantity(int64(1024), resource.BinarySI),
						mcmv1alpha1.ResourceNodes: *resource.NewQuantity(int64(3), resource.BinarySI),
					},
					Usage: corev1.ResourceList{
						corev1.ResourceCPU:       *resource.NewQuantity(int64(0), resource.BinarySI),
						corev1.ResourceMemory:    *resource.NewQuantity(int64(0), resource.BinarySI),
						corev1.ResourceStorage:   *resource.NewQuantity(int64(0), resource.BinarySI),
						mcmv1alpha1.ResourcePods: *resource.NewQuantity(int64(100), resource.BinarySI),
					},
				},
			},
			expected: []metav1.TableRow{{Cells: []interface{}{"clusterstatus1", "", "0/16", "0/1Ki", "0/1Ki", "3", "100", "0s", ""}}},
		},
	}

	for i, test := range tests {
		rows, err := printClusterStatus(&test.clusterStatus, printOptions)
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

func TestPrintClusterStatusList(t *testing.T) {
	tests := []struct {
		clusterStatusList mcm.ClusterStatusList
		expected          []metav1.TableRow
	}{
		{
			mcm.ClusterStatusList{
				Items: []mcm.ClusterStatus{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "clusterstatus1",
							Namespace:         "cn1",
							CreationTimestamp: metav1.Time{Time: time.Now().Add(1.9e9)},
						},
						Spec: mcm.ClusterStatusSpec{
							Capacity: corev1.ResourceList{
								corev1.ResourceCPU:        *resource.NewQuantity(int64(16), resource.BinarySI),
								corev1.ResourceMemory:     *resource.NewQuantity(int64(1024), resource.BinarySI),
								corev1.ResourceStorage:    *resource.NewQuantity(int64(1024), resource.BinarySI),
								mcmv1alpha1.ResourceNodes: *resource.NewQuantity(int64(3), resource.BinarySI),
							},
							Usage: corev1.ResourceList{
								corev1.ResourceCPU:       *resource.NewQuantity(int64(0), resource.BinarySI),
								corev1.ResourceMemory:    *resource.NewQuantity(int64(0), resource.BinarySI),
								corev1.ResourceStorage:   *resource.NewQuantity(int64(0), resource.BinarySI),
								mcmv1alpha1.ResourcePods: *resource.NewQuantity(int64(100), resource.BinarySI),
							},
						},
					},
				},
			},
			[]metav1.TableRow{{Cells: []interface{}{"clusterstatus1", "", "0/16", "0/1Ki", "0/1Ki", "3", "100", "0s", ""}}},
		},
	}

	for _, test := range tests {
		rows, err := printClusterStatusList(&test.clusterStatusList, printOptions)

		if err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			rows[i].Object.Object = nil
		}
		if !reflect.DeepEqual(test.expected, rows) {
			t.Errorf("mismatch: %s", diff.ObjectReflectDiff(test.expected, rows))
		}
	}
}

func TestPrintWorkSet(t *testing.T) {
	tests := []struct {
		workSet  mcm.WorkSet
		expected []metav1.TableRow
	}{
		// Basic cluster
		{
			workSet: mcm.WorkSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "workset1",
					Namespace: "default",
				},
				Spec: mcm.WorkSetSpec{
					Template: mcm.WorkTemplateSpec{
						Spec: mcm.WorkSpec{
							Type: mcm.ActionWorkType,
						},
					},
				},
				Status: mcm.WorkSetStatus{
					Status: mcm.WorkStatusType("running"),
				},
			},
			expected: []metav1.TableRow{{Cells: []interface{}{"workset1", "<none>", "running", "", "<unknown>"}}},
		},
	}

	for i, test := range tests {
		rows, err := printWorkSet(&test.workSet, printOptions)
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

func TestPrintWorkSetList(t *testing.T) {
	tests := []struct {
		workSetList mcm.WorkSetList
		expected    []metav1.TableRow
	}{
		{
			mcm.WorkSetList{
				Items: []mcm.WorkSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "workset1",
							Namespace: "default",
						},
						Spec: mcm.WorkSetSpec{
							Template: mcm.WorkTemplateSpec{
								Spec: mcm.WorkSpec{
									Type: mcm.ActionWorkType,
								},
							},
						},
						Status: mcm.WorkSetStatus{
							Status: mcm.WorkStatusType("running"),
						},
					},
				},
			},
			[]metav1.TableRow{{Cells: []interface{}{"workset1", "<none>", "running", "", "<unknown>"}}},
		},
	}

	for _, test := range tests {
		rows, err := printWorkSetList(&test.workSetList, printOptions)

		if err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			rows[i].Object.Object = nil
		}
		if !reflect.DeepEqual(test.expected, rows) {
			t.Errorf("mismatch: %s", diff.ObjectReflectDiff(test.expected, rows))
		}
	}
}

func TestPrintResourceView(t *testing.T) {
	tests := []struct {
		resourceView mcm.ResourceView
		expected     []metav1.TableRow
	}{
		// Basic cluster
		{
			resourceView: mcm.ResourceView{
				ObjectMeta: v1.ObjectMeta{
					Name: "resourceView1",
					Annotations: map[string]string{
						mcm.OwnersLabel: "resourceviews.v1beta1.mcm.ibm.com",
					},
				},
				Spec: mcm.ResourceViewSpec{
					Mode: mcm.PeriodicResourceUpdate,
					Scope: mcm.ViewFilter{
						Resource:  "pods",
						NameSpace: "ns1",
					},
				},
				Status: mcm.ResourceViewStatus{
					Conditions: []mcm.ViewCondition{
						{
							Type: mcm.WorkProcessing,
						},
					},
				},
			},
			expected: []metav1.TableRow{{Cells: []interface{}{"resourceView1", "<none>", "Processing", "", "<unknown>"}}},
		},
	}

	for i, test := range tests {
		rows, err := printResourceView(&test.resourceView, printOptions)
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

func TestPrintResourceViewList(t *testing.T) {
	tests := []struct {
		resourceViewList mcm.ResourceViewList
		expected         []metav1.TableRow
	}{
		{
			mcm.ResourceViewList{
				Items: []mcm.ResourceView{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "resourceView1",
							Annotations: map[string]string{
								mcm.OwnersLabel: "resourceviews.v1beta1.mcm.ibm.com",
							},
						},
						Spec: mcm.ResourceViewSpec{
							Mode: mcm.PeriodicResourceUpdate,
							Scope: mcm.ViewFilter{
								Resource:  "pods",
								NameSpace: "ns1",
							},
						},
						Status: mcm.ResourceViewStatus{
							Conditions: []mcm.ViewCondition{
								{
									Type: mcm.WorkProcessing,
								},
							},
						},
					},
				},
			},
			[]metav1.TableRow{{Cells: []interface{}{"resourceView1", "<none>", "Processing", "", "<unknown>"}}},
		},
	}

	for _, test := range tests {
		rows, err := printResourceViewList(&test.resourceViewList, printOptions)

		if err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			rows[i].Object.Object = nil
		}
		if !reflect.DeepEqual(test.expected, rows) {
			t.Errorf("mismatch: %s", diff.ObjectReflectDiff(test.expected, rows))
		}
	}
}

func TestPrintWork(t *testing.T) {
	tests := []struct {
		work     mcm.Work
		expected []metav1.TableRow
	}{
		// Basic cluster
		{
			work: mcm.Work{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "work1",
					Namespace: "work1",
				},
				Spec: mcm.WorkSpec{
					Type:  mcm.ResourceWorkType,
					Scope: mcm.ResourceFilter{},
				},
				Status: mcm.WorkStatus{
					Type: mcm.WorkCompleted,
				},
			},
			expected: []metav1.TableRow{{Cells: []interface{}{"work1", mcm.WorkType("Resource"), "", "Completed", "", "<unknown>"}}},
		},
	}

	for i, test := range tests {
		rows, err := printWork(&test.work, printOptions)
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

func TestPrintWorkList(t *testing.T) {
	tests := []struct {
		workList mcm.WorkList
		expected []metav1.TableRow
	}{
		{
			mcm.WorkList{
				Items: []mcm.Work{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "work1",
							Namespace: "work1",
						},
						Spec: mcm.WorkSpec{
							Type:  mcm.ResourceWorkType,
							Scope: mcm.ResourceFilter{},
						},
						Status: mcm.WorkStatus{
							Type: mcm.WorkCompleted,
						},
					},
				},
			},
			[]metav1.TableRow{{Cells: []interface{}{"work1", mcm.WorkType("Resource"), "", "Completed", "", "<unknown>"}}},
		},
	}

	for _, test := range tests {
		rows, err := printWorkList(&test.workList, printOptions)

		if err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			rows[i].Object.Object = nil
		}
		if !reflect.DeepEqual(test.expected, rows) {
			t.Errorf("mismatch: %s", diff.ObjectReflectDiff(test.expected, rows))
		}
	}
}

var clientCertUsage = []certificates.KeyUsage{
	certificates.UsageDigitalSignature,
	certificates.UsageKeyEncipherment,
	certificates.UsageClientAuth,
}

func TestPrintClusterJoinRequest(t *testing.T) {
	tests := []struct {
		clusterJoinRequest mcm.ClusterJoinRequest
		expected           []metav1.TableRow
	}{
		// Basic cluster
		{
			clusterJoinRequest: mcm.ClusterJoinRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: "acmjoinName",
				},
				Spec: mcm.ClusterJoinRequestSpec{
					ClusterName:      "clustername1",
					ClusterNamespace: "clusternamespace1",
					CSR: certificates.CertificateSigningRequestSpec{
						Request: []byte(""),
						Usages:  clientCertUsage,
					},
				},
			},
			expected: []metav1.TableRow{{Cells: []interface{}{"acmjoinName", "clustername1", "clusternamespace1", "Pending", "<unknown>"}}},
		},
	}

	for i, test := range tests {
		rows, err := printHCMJoin(&test.clusterJoinRequest, printOptions)
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

func TestPrintClusterJoinRequestList(t *testing.T) {
	tests := []struct {
		clusterJoinRequestList mcm.ClusterJoinRequestList
		expected               []metav1.TableRow
	}{
		{
			mcm.ClusterJoinRequestList{
				Items: []mcm.ClusterJoinRequest{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "acmjoinName",
						},
						Spec: mcm.ClusterJoinRequestSpec{
							ClusterName:      "clustername1",
							ClusterNamespace: "clusternamespace1",
							CSR: certificates.CertificateSigningRequestSpec{
								Request: []byte(""),
								Usages:  clientCertUsage,
							},
						},
					},
				},
			},
			[]metav1.TableRow{{Cells: []interface{}{"acmjoinName", "clustername1", "clusternamespace1", "Pending", "<unknown>"}}},
		},
	}

	for _, test := range tests {
		rows, err := printHCMJoinList(&test.clusterJoinRequestList, printOptions)
		if err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			rows[i].Object.Object = nil
		}
		if !reflect.DeepEqual(test.expected, rows) {
			t.Errorf("mismatch: %s", diff.ObjectReflectDiff(test.expected, rows))
		}
	}
}
