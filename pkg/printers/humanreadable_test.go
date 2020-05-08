// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package printers

import (
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
)

var printOptions = PrintOptions{
	WithNamespace: true,
	ShowLabels:    true,
}

var columns = []metav1.TableColumnDefinition{
	{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
	{Name: "Ready", Type: "string", Description: "The aggregate readiness state of this pod for accepting traffic."},
	{Name: "Status", Type: "string", Description: "The aggregate status of the containers in this pod."},
	{Name: "Restarts", Type: "integer", Description: "The number of times the containers in this pod have been restarted."},
	{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
}
var columns1 = []metav1.TableColumnDefinition{
	{Name: "Name1", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
	{Name: "Ready", Type: "string", Description: "The aggregate readiness state of this pod for accepting traffic."},
}

var table = &metav1.Table{
	ColumnDefinitions: columns,
	Rows: []metav1.TableRow{
		{Cells: []interface{}{"foo", "1/2", "Pending", int64(10), "370d", "10.1.2.3", "test-node", "nominated-node", "1/2"}, Object: runtime.RawExtension{}},
	},
}
var pod1 = &api.Pod{
	ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo", CreationTimestamp: metav1.NewTime(time.Now().Add(-370 * 24 * time.Hour))},
	Spec: api.PodSpec{
		Containers: []api.Container{
			{Name: "ctr1"},
			{Name: "ctr2", Ports: []api.ContainerPort{{ContainerPort: 9376}}},
		},
		NodeName: "test-node",
		ReadinessGates: []api.PodReadinessGate{
			{
				ConditionType: api.PodConditionType("condition1"),
			},
			{
				ConditionType: api.PodConditionType("condition2"),
			},
		},
	},
	Status: api.PodStatus{
		Conditions: []api.PodCondition{
			{
				Type:   api.PodConditionType("condition1"),
				Status: api.ConditionFalse,
			},
			{
				Type:   api.PodConditionType("condition2"),
				Status: api.ConditionTrue,
			},
		},
		PodIPs: []api.PodIP{{IP: "10.1.2.3"}},
		Phase:  api.PodPending,
		ContainerStatuses: []api.ContainerStatus{
			{Name: "ctr1", State: api.ContainerState{Running: &api.ContainerStateRunning{}}, RestartCount: 10, Ready: true},
			{Name: "ctr2", State: api.ContainerState{Waiting: &api.ContainerStateWaiting{}}, RestartCount: 0},
		},
		NominatedNodeName: "nominated-node",
	},
}

var table1 = &metav1.Table{
	ColumnDefinitions: columns,
	Rows: []metav1.TableRow{
		{
			Cells:  []interface{}{"foo", "1/2", "Pending", int64(10), "370d", "10.1.2.3", "test-node", "nominated-node", "1/2"},
			Object: runtime.RawExtension{Object: pod1},
		},
	},
}

func newResourceViewList() runtime.Object {
	return &mcm.ResourceViewList{
		Items: []mcm.ResourceView{
			{
				ObjectMeta: metav1.ObjectMeta{
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
	}
}
func newResourceView(table *metav1.Table) runtime.Object {
	raw, err := json.Marshal(table)
	if err != nil {
		panic(err)
	}
	return &mcm.ResourceView{
		ObjectMeta: metav1.ObjectMeta{
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
			SummaryOnly: true,
		},
		Status: mcm.ResourceViewStatus{
			Conditions: []mcm.ViewCondition{
				{
					Type: mcm.WorkProcessing,
				},
			},
			Results: map[string]runtime.RawExtension{
				"c1": {
					Raw: raw,
				},
			},
		},
	}
}

func TestNewTablePrinter(t *testing.T) {
	newHumanReadablePrinter := NewHumanReadablePrinter(nil, printOptions)
	newHumanReadablePrinter.AddTabWriter(true)
	newTablePrinter := NewTablePrinter()
	newTablePrinter.AddTabWriter(true)
	newHumanReadablePrinter.EnsurePrintHeaders()

	table := newHumanReadablePrinter.formatTable(columns1, table)
	if len(table) == 0 {
		t.Errorf("Table length is: %v", table)
	}
}
func defaultPrintFunc(obj *mcm.ResourceView, w io.Writer, options PrintOptions) error {
	return nil
}
func defaultPrintFunc2(obj *mcm.ResourceView, options PrintOptions) ([]metav1beta1.TableRow, error) {
	return []metav1beta1.TableRow{}, nil
}

func TestHandler(t *testing.T) {
	newHumanReadablePrinter := NewHumanReadablePrinter(nil, printOptions)
	strings1 := []string{"s1", "s2"}
	strings2 := []string{"s4", "s3"}
	resourceView1 := newResourceView(table1)
	err := newHumanReadablePrinter.Handler(strings1, strings2, defaultPrintFunc)
	if err != nil {
		t.Errorf("Table handler error:%v", err)
	}
	err = newHumanReadablePrinter.TableHandler(columns1, defaultPrintFunc)
	if err == nil {
		t.Errorf("Table handler should have error")
	}
	err = newHumanReadablePrinter.DefaultTableHandler(columns1, defaultPrintFunc)
	if err == nil {
		t.Errorf("Default table handler should have error:%v", err)
	}
	str := newHumanReadablePrinter.HandledResources()
	if len(str) != 1 {
		t.Errorf("Handler resources error")
	}
	_, err = newHumanReadablePrinter.PrintTable(resourceView1, printOptions)
	if err != nil {
		t.Errorf("Print table error:%v", err)
	}

	emptyPrintOption := PrintOptions{}

	err = newHumanReadablePrinter.Handler(strings1, strings2, defaultPrintFunc2)
	if err == nil {
		t.Errorf("Table handler should have error:%v", err)
	}
	err = newHumanReadablePrinter.TableHandler(columns1, defaultPrintFunc2)
	if err == nil {
		t.Errorf("Table handler should have error")
	}
	err = newHumanReadablePrinter.DefaultTableHandler(columns1, defaultPrintFunc2)
	if err != nil {
		t.Errorf("Default table handler error:%v", err)
	}
	nrvl := newResourceViewList()
	_, err = newHumanReadablePrinter.PrintTable(nrvl, emptyPrintOption)
	if err == nil {
		t.Errorf("Print table should have error:%v", err)
	}
}

func TestDecorateTable(t *testing.T) {
	err := DecorateTable(table1, printOptions)
	if err != nil {
		t.Errorf("Decorate table error:%v", err)
	}
}

func TestFormatResourceName(t *testing.T) {
	type args struct {
		kind     schema.GroupKind
		name     string
		withKind bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "case1", args: args{name: "name1", kind: schema.GroupKind{Group: "acm", Kind: "kind"}, withKind: true}, want: "kind.acm/name1"},
		{name: "case1", args: args{name: "name1", kind: schema.GroupKind{Group: "acm", Kind: "kind"}, withKind: false}, want: "name1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatResourceName(tt.args.kind, tt.args.name, tt.args.withKind); got != tt.want {
				t.Errorf("FormatResourceName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendLabels(t *testing.T) {
	type args struct {
		itemLabels   map[string]string
		columnLabels []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "case1", args: args{itemLabels: map[string]string{"m1": "key1", "m2": "key2"}, columnLabels: []string{"m1", "s2"}}, want: "	key1	<none>"},
		{name: "case1", args: args{itemLabels: map[string]string{"m1": "key1", "m2": "key2"}, columnLabels: []string{"s1", "m2"}}, want: "	<none>	key2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AppendLabels(tt.args.itemLabels, tt.args.columnLabels); got != tt.want {
				t.Errorf("AppendLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendAllLabels(t *testing.T) {
	type args struct {
		showLabels bool
		itemLabels map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "case1", args: args{itemLabels: map[string]string{"m1": "key1", "m2": "key2"}, showLabels: true}, want: "	m1=key1,m2=key2\n"},
		{name: "case1", args: args{itemLabels: map[string]string{"m1": "key1", "m2": "key2"}, showLabels: false}, want: "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AppendAllLabels(tt.args.showLabels, tt.args.itemLabels); got != tt.want {
				t.Errorf("AppendAllLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestPrintDefault(t *testing.T) {
	raw, err := json.Marshal(table)
	if err != nil {
		panic(err)
	}
	printUnstructured(raw)
	resourceviewlist := newResourceViewList()
	_, err = printObjectMeta(resourceviewlist)
	if err != nil {
		t.Errorf("print Object Meta error:%v", err)
	}
}

func TestValidateRowPrintHandlerFunc(t *testing.T) {
	type args struct {
		printFunc reflect.Value
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"case1", args{reflect.ValueOf(defaultPrintFunc)}, true},
		{"case2", args{reflect.ValueOf(defaultPrintFunc2)}, false},
		{"case3", args{reflect.ValueOf(printObjectMeta)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateRowPrintHandlerFunc(tt.args.printFunc); (err != nil) != tt.wantErr {
				t.Errorf("ValidateRowPrintHandlerFunc() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePrintHandlerFunc(t *testing.T) {
	type args struct {
		printFunc reflect.Value
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"case1", args{reflect.ValueOf(defaultPrintFunc)}, false},
		{"case2", args{reflect.ValueOf(defaultPrintFunc2)}, true},
		{"case3", args{reflect.ValueOf(printObjectMeta)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePrintHandlerFunc(tt.args.printFunc); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrintHandlerFunc() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
