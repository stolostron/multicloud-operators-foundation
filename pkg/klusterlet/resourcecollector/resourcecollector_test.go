package resourcecollector

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

	fakecluster "github.com/open-cluster-management/api/client/cluster/clientset/versioned/fake"
	clusterapiv1 "github.com/open-cluster-management/api/cluster/v1"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	fakekube "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	certutil "k8s.io/client-go/util/cert"
)

func TestReconcile(t *testing.T) {
	handler := &testHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	ioutil.WriteFile("/tmp/token", []byte("test"), 0644)
	defer os.Remove("/tmp/token")

	ca, _, _ := certutil.GenerateSelfSignedCertKey(server.URL, []net.IP{}, []string{})
	cases := []struct {
		name                   string
		resources              []runtime.Object
		existingCapacity       clusterapiv1.ResourceList
		prometheusData         model.Vector
		validateClusterActions func(t *testing.T, actions []clienttesting.Action)
		validateKubeActions    func(t *testing.T, actions []clienttesting.Action)
	}{
		{
			name:      "missing configmap",
			resources: []runtime.Object{},
			existingCapacity: clusterapiv1.ResourceList{
				"cpu_worker": *resource.NewQuantity(0, resource.DecimalSI),
			},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "create")
			},
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			prometheusData: model.Vector{},
		},
		{
			name:             "missing configmap having node",
			resources:        []runtime.Object{newNode("node1", 1, true)},
			existingCapacity: clusterapiv1.ResourceList{},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "create")
			},
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "update")
				cluster := actions[1].(clienttesting.UpdateAction).GetObject().(*clusterapiv1.ManagedCluster)
				if !cluster.Status.Capacity[resourceCPUWorker].Equal(*resource.NewQuantity(1, resource.DecimalSI)) {
					t.Errorf("Expect cpu worker capcity is not correct")
				}
			},
			prometheusData: model.Vector{},
		},
		{
			name:             "no updates with same capacity",
			resources:        []runtime.Object{newNode("node1", 1, true)},
			existingCapacity: clusterapiv1.ResourceList{"cpu_worker": *resource.NewQuantity(1, resource.DecimalSI)},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "create")
			},
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			prometheusData: model.Vector{},
		},
		{
			name:      "missing ca",
			resources: []runtime.Object{newConfigmap("")},
			existingCapacity: clusterapiv1.ResourceList{
				"cpu_worker": *resource.NewQuantity(0, resource.DecimalSI),
			},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			prometheusData: model.Vector{},
		},
		{
			name:             "no updates with same capacity",
			resources:        []runtime.Object{newNode("node1", 1, true)},
			existingCapacity: clusterapiv1.ResourceList{"cpu_worker": *resource.NewQuantity(1, resource.DecimalSI)},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "create")
			},
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			prometheusData: model.Vector{},
		},
		{
			name:      "missing node metrics",
			resources: []runtime.Object{newConfigmap(string(ca))},
			existingCapacity: clusterapiv1.ResourceList{
				"cpu_worker": *resource.NewQuantity(0, resource.DecimalSI),
			},
			validateKubeActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
			prometheusData: model.Vector{},
		},
		{
			name:      "missing node",
			resources: []runtime.Object{newConfigmap(string(ca))},
			existingCapacity: clusterapiv1.ResourceList{
				"cpu_worker": *resource.NewQuantity(0, resource.DecimalSI),
			},
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
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get")
			},
		},
		{
			name:             "update status",
			resources:        []runtime.Object{newConfigmap(string(ca)), newNode("node1", 1, true)},
			existingCapacity: clusterapiv1.ResourceList{},
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
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "update")
				cluster := actions[1].(clienttesting.UpdateAction).GetObject().(*clusterapiv1.ManagedCluster)
				if !cluster.Status.Capacity[resourceCPUWorker].Equal(*resource.NewQuantity(1, resource.DecimalSI)) {
					t.Errorf("Expect cpu worker capcity is not correct")
				}
				if !cluster.Status.Capacity[resourceSocket].Equal(*resource.NewQuantity(1, resource.DecimalSI)) {
					t.Errorf("Expect socket capcity is not correct")
				}
				if !cluster.Status.Capacity[resourceSocketWorker].Equal(*resource.NewQuantity(1, resource.DecimalSI)) {
					t.Errorf("Expect worker socket capcity is not correct")
				}
			},
		},
		{
			name:             "update status with worker/master",
			resources:        []runtime.Object{newConfigmap(string(ca)), newNode("node1", 2, true), newNode("node2", 1, false)},
			existingCapacity: clusterapiv1.ResourceList{},
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
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "update")
				cluster := actions[1].(clienttesting.UpdateAction).GetObject().(*clusterapiv1.ManagedCluster)
				if !cluster.Status.Capacity[resourceCPUWorker].Equal(*resource.NewQuantity(2, resource.DecimalSI)) {
					t.Errorf("Expect cpu worker capcity is not correct")
				}
				if !cluster.Status.Capacity[resourceSocket].Equal(*resource.NewQuantity(3, resource.DecimalSI)) {
					t.Errorf("Expect socket capcity is not correct")
				}
				if !cluster.Status.Capacity[resourceSocketWorker].Equal(*resource.NewQuantity(2, resource.DecimalSI)) {
					t.Errorf("Expect worker socket capcity is not correct")
				}
			},
		},
		{
			name:             "update status with nil apcity",
			resources:        []runtime.Object{newConfigmap(string(ca)), newNode("node1", 1, true)},
			existingCapacity: nil,
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
			validateClusterActions: func(t *testing.T, actions []clienttesting.Action) {
				assertActions(t, actions, "get", "update")
				cluster := actions[1].(clienttesting.UpdateAction).GetObject().(*clusterapiv1.ManagedCluster)
				if !cluster.Status.Capacity[resourceCPUWorker].Equal(*resource.NewQuantity(1, resource.DecimalSI)) {
					t.Errorf("Expect cpu worker capcity is not correct")
				}
				if !cluster.Status.Capacity[resourceSocket].Equal(*resource.NewQuantity(1, resource.DecimalSI)) {
					t.Errorf("Expect socket capcity is not correct")
				}
				if !cluster.Status.Capacity[resourceSocketWorker].Equal(*resource.NewQuantity(1, resource.DecimalSI)) {
					t.Errorf("Expect worker socket capcity is not correct")
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakekubeClient := fakekube.NewSimpleClientset(c.resources...)
			cluster := &clusterapiv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster1",
				},
				Status: clusterapiv1.ManagedClusterStatus{
					Capacity: c.existingCapacity,
				},
			}
			infomerFactory := informers.NewSharedInformerFactory(fakekubeClient, 10*time.Minute)
			handler.val = c.prometheusData
			fakeclusterClient := fakecluster.NewSimpleClientset(cluster)

			store := infomerFactory.Core().V1().Nodes().Informer().GetStore()
			for _, obj := range c.resources {
				if _, ok := obj.(*corev1.Node); ok {
					store.Add(obj)
				}
			}
			ctrl := &resourceCollector{
				nodeLister:         infomerFactory.Core().V1().Nodes().Lister(),
				kubeClient:         fakekubeClient,
				clusterClient:      fakeclusterClient,
				clusterName:        "cluster1",
				server:             server.URL,
				tokenFile:          "/tmp/token",
				componentNamespace: "default",
			}

			ctrl.reconcile(context.TODO())
			c.validateClusterActions(t, fakeclusterClient.Actions())
			c.validateKubeActions(t, fakekubeClient.Actions())
		})
	}
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
