package nodecollector

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	clusterapiv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clienttesting "k8s.io/client-go/testing"
	certutil "k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcile(t *testing.T) {
	handler := &testHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	ioutil.WriteFile("/tmp/token", []byte("test"), 0644)
	defer os.Remove("/tmp/token")

	s := scheme.Scheme
	s.AddKnownTypes(clusterv1beta1.GroupVersion, &clusterv1beta1.ManagedClusterInfo{})

	ca, _, _ := certutil.GenerateSelfSignedCertKey(server.URL, []net.IP{}, []string{})
	cases := []struct {
		name                string
		resources           []runtime.Object
		existingNodeList    []clusterv1beta1.NodeStatus
		prometheusData      model.Vector
		expectedNodeList    []clusterv1beta1.NodeStatus
		validateKubeActions func(t *testing.T, actions []clienttesting.Action)
	}{
		{
			name:             "missing configmap",
			resources:        []runtime.Object{},
			existingNodeList: []clusterv1beta1.NodeStatus{},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "create")
			},
			expectedNodeList: []clusterv1beta1.NodeStatus{},
			prometheusData:   model.Vector{},
		},
		{
			name:             "missing configmap having node",
			resources:        []runtime.Object{newNode("node1", 1, true)},
			existingNodeList: []clusterv1beta1.NodeStatus{},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "create")
			},
			expectedNodeList: []clusterv1beta1.NodeStatus{
				newResourceStatus("node1", map[clusterapiv1.ResourceName]int64{"cpu": 1}, true),
			},
			prometheusData: model.Vector{},
		},
		{
			name:             "no updates with same capacity",
			resources:        []runtime.Object{newNode("node1", 1, true)},
			existingNodeList: []clusterv1beta1.NodeStatus{newResourceStatus("node1", map[clusterapiv1.ResourceName]int64{"cpu": 1}, true)},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "create")
			},
			expectedNodeList: []clusterv1beta1.NodeStatus{
				newResourceStatus("node1", map[clusterapiv1.ResourceName]int64{"cpu": 1}, true),
			},
			prometheusData: model.Vector{},
		},
		{
			name:             "missing ca",
			resources:        []runtime.Object{newConfigmap("")},
			existingNodeList: []clusterv1beta1.NodeStatus{newResourceStatus("node1", map[clusterapiv1.ResourceName]int64{"cpu": 1}, true)},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			expectedNodeList: []clusterv1beta1.NodeStatus{},
			prometheusData:   model.Vector{},
		},
		{
			name:             "no updates with same capacity",
			resources:        []runtime.Object{newNode("node1", 2, false)},
			existingNodeList: []clusterv1beta1.NodeStatus{newResourceStatus("node1", map[clusterapiv1.ResourceName]int64{"cpu": 1}, true)},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "create")
			},
			expectedNodeList: []clusterv1beta1.NodeStatus{
				newResourceStatus("node1", map[clusterapiv1.ResourceName]int64{"cpu": 2}, false),
			},
			prometheusData: model.Vector{},
		},
		{
			name:             "missing node metrics",
			resources:        []runtime.Object{newConfigmap(string(ca))},
			existingNodeList: []clusterv1beta1.NodeStatus{},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			expectedNodeList: []clusterv1beta1.NodeStatus{},
			prometheusData:   model.Vector{},
		},
		{
			name:             "missing node",
			resources:        []runtime.Object{newConfigmap(string(ca))},
			existingNodeList: []clusterv1beta1.NodeStatus{},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			prometheusData: model.Vector{
				&model.Sample{
					Value: 1,
					Metric: model.Metric{
						"node": "node1",
					},
				},
			},
			expectedNodeList: []clusterv1beta1.NodeStatus{},
		},
		{
			name:             "update status",
			resources:        []runtime.Object{newConfigmap(string(ca)), newNode("node1", 1, true)},
			existingNodeList: []clusterv1beta1.NodeStatus{newResourceStatus("node1", map[clusterapiv1.ResourceName]int64{"cpu": 1}, true)},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			prometheusData: model.Vector{
				&model.Sample{
					Value: 1,
					Metric: model.Metric{
						"node": "node1",
					},
				},
			},
			expectedNodeList: []clusterv1beta1.NodeStatus{
				newResourceStatus("node1", map[clusterapiv1.ResourceName]int64{"cpu": 1, "core": 1, "socket": 1}, true),
			},
		},
		{
			name:             "update status with worker/master",
			resources:        []runtime.Object{newConfigmap(string(ca)), newNode("node1", 2, true), newNode("node2", 1, false)},
			existingNodeList: []clusterv1beta1.NodeStatus{},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			prometheusData: model.Vector{
				&model.Sample{
					Value: 2,
					Metric: model.Metric{
						"node": "node1",
					},
				},
				&model.Sample{
					Value: 1,
					Metric: model.Metric{
						"node": "node2",
					},
				},
			},
			expectedNodeList: []clusterv1beta1.NodeStatus{
				newResourceStatus("node1", map[clusterapiv1.ResourceName]int64{"cpu": 2, "core": 2, "socket": 2}, true),
				newResourceStatus("node2", map[clusterapiv1.ResourceName]int64{"cpu": 1, "core": 1, "socket": 1}, false),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakekubeClient := fakekube.NewSimpleClientset(c.resources...)
			clusterinfo := &clusterv1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster1",
					Namespace: "cluster1",
				},
				Status: clusterv1beta1.ClusterInfoStatus{
					NodeList: c.existingNodeList,
				},
			}
			infomerFactory := informers.NewSharedInformerFactory(fakekubeClient, 10*time.Minute)
			handler.val = c.prometheusData
			client := fake.NewClientBuilder().WithScheme(s).WithObjects(clusterinfo).Build()

			store := infomerFactory.Core().V1().Nodes().Informer().GetStore()
			for _, obj := range c.resources {
				if _, ok := obj.(*corev1.Node); ok {
					store.Add(obj)
				}
			}
			ctrl := &resourceCollector{
				nodeLister:         infomerFactory.Core().V1().Nodes().Lister(),
				kubeClient:         fakekubeClient,
				client:             client,
				clusterName:        "cluster1",
				server:             server.URL,
				tokenFile:          "/tmp/token",
				componentNamespace: "default",
			}

			ctrl.reconcile(context.TODO())
			actualInfo := &clusterv1beta1.ManagedClusterInfo{}
			err := client.Get(context.TODO(), types.NamespacedName{Namespace: "cluster1", Name: "cluster1"}, actualInfo)
			if err != nil {
				t.Errorf("expected no error: %v", err)
			}
			if !apiequality.Semantic.DeepEqual(actualInfo.Status.NodeList, c.expectedNodeList) {
				t.Errorf("unexpected nodelist: %v, %v", actualInfo.Status.NodeList, c.expectedNodeList)
			}

			c.validateKubeActions(t, fakekubeClient.Actions())
		})
	}
}

func newResourceStatus(name string, resources map[clusterapiv1.ResourceName]int64, isworker bool) clusterv1beta1.NodeStatus {
	status := clusterv1beta1.NodeStatus{
		Name:     name,
		Capacity: clusterv1beta1.ResourceList{},
		Conditions: []clusterv1beta1.NodeCondition{
			{
				Type: corev1.NodeReady,
			},
		},
	}

	if isworker {
		status.Labels = map[string]string{
			"node-role.kubernetes.io/worker": "",
		}
	}

	for k, v := range resources {
		status.Capacity[k] = *resource.NewQuantity(v, resource.DecimalSI)
	}

	return status
}

func newNode(name string, cpu int64, isworker bool) *corev1.Node {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				"cpu": *resource.NewQuantity(cpu, resource.DecimalSI),
			},
		},
	}

	if isworker {
		node.Labels = map[string]string{
			"node-role.kubernetes.io/worker": "",
		}
	}
	return node
}

func newConfigmap(ca string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caConfigMapName,
			Namespace: "default",
		},
		Data: map[string]string{
			"service-ca.crt": ca,
		},
	}
}

type testHandler struct {
	val model.Vector
}

func (t *testHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	data, _ := json.Marshal(t.val)
	v := struct {
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
	}{
		Type:   model.ValVector,
		Result: data,
	}
	resData, _ := json.Marshal(v)
	apiResponse := struct {
		Status    string                 `json:"status"`
		Data      json.RawMessage        `json:"data"`
		ErrorType prometheusv1.ErrorType `json:"errorType"`
		Error     string                 `json:"error"`
		Warnings  []string               `json:"warnings,omitempty"`
	}{
		Status:   "success",
		Data:     resData,
		Warnings: []string{},
	}

	responseData, _ := json.Marshal(apiResponse)
	resp.Write(responseData)
}

func assertActions(t *testing.T, actualActions []clienttesting.Action, expectedVerbs ...string) {
	if len(actualActions) != len(expectedVerbs) {
		t.Fatalf("expected %d call but got: %#v", len(expectedVerbs), actualActions)
	}
	for i, expected := range expectedVerbs {
		if actualActions[i].GetVerb() != expected {
			t.Errorf("expected %s action but got: %#v", expected, actualActions[i])
		}
	}
}

// AssertNoActions asserts no actions are happened
func assertNoActions(t *testing.T, actualActions []clienttesting.Action) {
	assertActions(t, actualActions)
}
