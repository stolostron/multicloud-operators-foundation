// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterlet

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1beta1"
	hcmfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	clusterfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset/fake"
	informers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	helmutil "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/helm"
	corev1 "k8s.io/api/core/v1"
	extensionv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/helm/pkg/helm"

	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"
	routev1Fake "github.com/openshift/client-go/route/clientset/versioned/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type testKlusterlet struct {
	*Klusterlet
	workStore         cache.Store
	kubeControl       *restutils.FakeKubeControl
	fakeHCMClient     *hcmfake.Clientset
	fakeClusterClient *clusterfake.Clientset
}

var (
	alwaysReady = func() bool { return true }

	kubeNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
	}

	apiConfig = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "platform-api",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"KUBERNETES_API_EXTERNAL_URL": "https://127.0.0.1:8001",
			"CLUSTER_EXTERNAL_URL":        "https://127.0.0.1:8443",
		},
	}

	kubeEndpoints = &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "127.0.0.1",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port: 443,
					},
				},
			},
		},
	}

	kubeMonitoringService = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitoring",
			Namespace: "kube-system",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
	}

	kubeMonitoringSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitoring",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"tls.crt": []byte("aaa"),
			"tls.key": []byte("aaa"),
		},
	}

	klusterletIngress = &extensionv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extension/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "mcm-ingress-testcluster-klusterlet",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: extensionv1beta1.IngressSpec{
			Rules: []extensionv1beta1.IngressRule{
				{
					Host: "test.com",
				},
			},
		},
		Status: extensionv1beta1.IngressStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: "127.0.0.1",
					},
				},
			},
		},
	}

	klusterletService = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "klusterlet",
			Namespace: "kube-system",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: "10.0.0.1",
					},
				},
			},
		},
	}
)

func newTestKlusterlet(configMap *corev1.ConfigMap, clusterRegistry *clusterv1alpha1.Cluster) (
	*testKlusterlet, *helm.FakeClient) {
	fakeKubeClient := kubefake.NewSimpleClientset(
		configMap, kubeNode, kubeEndpoints,
		kubeMonitoringService, kubeMonitoringSecret,
		klusterletIngress, klusterletService)
	fakeHubKubeClient := kubefake.NewSimpleClientset()
	fakeRouteV1Client := routev1Fake.NewSimpleClientset()
	fakehcmClient := hcmfake.NewSimpleClientset()
	clusterFakeClient := clusterfake.NewSimpleClientset(clusterRegistry)
	helmclient := &helm.FakeClient{}
	helmcontrol := helmutil.NewFakeHelmControl(helmclient)

	informerFactory := informers.NewSharedInformerFactory(fakehcmClient, time.Minute*10)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClient, time.Minute*10)
	fakekubecontrol := restutils.NewFakeKubeControl()

	config := &Config{
		ClusterName:        "testcluster",
		ClusterNamespace:   "test",
		ClusterLabels:      map[string]string{},
		ClusterAnnotations: map[string]string{},
		KlusterletIngress:  "kube-system/mcm-ingress-testcluster-klusterlet",
		KlusterletService:  "kube-system/klusterlet",
	}

	klusterlet := NewKlusterlet(
		config, fakeKubeClient, fakeRouteV1Client, fakehcmClient, nil,
		fakeHubKubeClient, clusterFakeClient, helmcontrol, fakekubecontrol, kubeInformerFactory, informerFactory, nil)

	klusterlet.nodeSynced = alwaysReady
	klusterlet.podSynced = alwaysReady
	klusterlet.workSynced = alwaysReady

	return &testKlusterlet{
		klusterlet,
		informerFactory.Mcm().V1beta1().Works().Informer().GetStore(),
		fakekubecontrol,
		fakehcmClient,
		clusterFakeClient,
	}, helmclient
}

func newCluster() *clusterv1alpha1.Cluster {
	return &clusterv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: "test",
			Labels:    map[string]string{},
		},
		Spec: clusterv1alpha1.ClusterSpec{
			KubernetesAPIEndpoints: clusterv1alpha1.KubernetesAPIEndpoints{
				ServerEndpoints: []clusterv1alpha1.ServerAddressByClientCIDR{
					{
						ServerAddress: "127.0.0.1:8001",
					},
				},
			},
		},
		Status: clusterv1alpha1.ClusterStatus{
			Conditions: []clusterv1alpha1.ClusterCondition{
				{
					Type:   clusterv1alpha1.ClusterOK,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
}

func newWork(name string, workType v1beta1.WorkType) *v1beta1.Work {
	work := &v1beta1.Work{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Spec: v1beta1.WorkSpec{
			Cluster: corev1.LocalObjectReference{
				Name: "testcluster",
			},
			Type: workType,
		},
	}

	if workType == v1beta1.ResourceWorkType {
		work.Spec.Scope = v1beta1.ResourceFilter{
			ResourceType: "jobs",
		}
	}

	return work
}

func syncWork(t *testing.T, manager *testKlusterlet, work *v1beta1.Work) {
	key, err := cache.MetaNamespaceKeyFunc(work)
	if err != nil {
		t.Errorf("Could not get key for daemon.")
	}
	manager.processWork(key)
}

func TestSyncClusterStatus(t *testing.T) {
	//Check if the cluster in cluster registry is created once the cluster is first created.
	klusterlet, _ := newTestKlusterlet(apiConfig, &clusterv1alpha1.Cluster{})
	hcmclient := klusterlet.fakeHCMClient
	clusterclient := klusterlet.fakeClusterClient
	klusterlet.syncClusterStatus()

	var updateCount, createCount int
	for _, action := range clusterclient.Actions() {
		if action.Matches("update", "clusters") {
			updateCount++
		}
		if action.Matches("create", "clusters") {
			createCount++
		}
	}
	if updateCount != 1 {
		t.Errorf("syncClusterStatus() clusterToUpdate = %v, want %v", updateCount, 1)
	}
	if createCount != 1 {
		t.Errorf("syncClusterStatus() clusterToCreate = %v, want %v", createCount, 1)
	}

	createCount = 0
	for _, action := range hcmclient.Actions() {
		if action.Matches("create", "clusterstatuses") {
			createCount++
		}
	}
	if createCount != 1 {
		t.Errorf("syncClusterStatus() clusterstatusToCreate = %v, want %v", createCount, 1)
	}

	//Check if cluster registry is updated twice when the the cluster is refreshed.
	var refreshedAPIConfig = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "platform-api",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"KUBERNETES_API_EXTERNAL_URL": "https://127.0.1.1:8001",
		},
	}

	klusterlet, _ = newTestKlusterlet(refreshedAPIConfig, newCluster())
	clusterclient = klusterlet.fakeClusterClient
	klusterlet.syncClusterStatus()

	updateCount = 0
	createCount = 0
	for _, action := range clusterclient.Actions() {
		if action.Matches("update", "clusters") {
			updateCount++
		}
		if action.Matches("create", "clusters") {
			createCount++
		}
	}
	if updateCount != 2 {
		t.Errorf("syncClusterStatus() clusterToUpdate = %v, want %v", updateCount, 2)
	}
	if createCount != 0 {
		t.Errorf("syncClusterStatus() clusterToCreate = %v, want %v", createCount, 0)
	}
}

func TestReadKlusterletConfig(t *testing.T) {
	klusterlet, _ := newTestKlusterlet(apiConfig, newCluster())
	//Test setting klusterlet address as IP
	klusterlet.config.KlusterletAddress = "192.168.0.1"
	endpoint, _, _ := klusterlet.readKlusterletConfig()
	if endpoint.IP != "192.168.0.1" {
		t.Errorf("endpoint IP = %v, want %v", endpoint.IP, "192.168.0.1")
	}

	if endpoint.Hostname != "test.com" {
		t.Errorf("endpoint hostname = %v, want %v", endpoint.Hostname, "test.com")
	}

	//Test setting klusterlet address as hostName
	klusterlet.config.KlusterletAddress = "test.klusterlet"
	endpoint, _, _ = klusterlet.readKlusterletConfig()
	if endpoint.IP != klusterletIngress.Status.LoadBalancer.Ingress[0].IP && endpoint.Hostname != "test.klusterlet" {
		t.Errorf("endpoint host = %v, want %v", endpoint.Hostname, "test.klusterlet")
	}

	klusterlet.config.KlusterletIngress = ""
	klusterlet.config.KlusterletAddress = ""
	endpoint, port, _ := klusterlet.readKlusterletConfig()
	wantIP, wantPort := klusterletService.Status.LoadBalancer.Ingress[0].IP, klusterletService.Spec.Ports[0].Port
	if endpoint.IP != wantIP && port.Port != wantPort {
		t.Errorf("endpoint ip, port = %v %v, want %v, %v", endpoint.IP, port.Port, wantIP, wantPort)
	}
}

func TestProcessWork(t *testing.T) {
	work := newWork("work1", v1beta1.ResourceWorkType)
	klusterlet, _ := newTestKlusterlet(apiConfig, &clusterv1alpha1.Cluster{})
	hcmclient := klusterlet.fakeHCMClient
	klusterlet.workStore.Add(work)
	syncWork(t, klusterlet, work)

	var updateCount int
	for _, action := range hcmclient.Actions() {
		if action.Matches("update", "works") {
			updateCount++
		}
	}

	if updateCount != 0 {
		t.Errorf("processWork() workToUpdate = %v, want %v", updateCount, 0)
	}

	work1 := newWork("work1", v1beta1.ResourceWorkType)
	work1.Status.Type = v1beta1.WorkCompleted
	klusterlet.workStore.Add(work1)
	syncWork(t, klusterlet, work1)
	for _, action := range hcmclient.Actions() {
		if action.Matches("update", "works") {
			updateCount++
		}
	}
	if updateCount != 0 {
		t.Errorf("processWork() workToUpdate = %v, want %v", updateCount, 0)
	}

	work2 := newWork("work1", v1beta1.ResourceWorkType)
	work2.Status.Type = v1beta1.WorkFailed
	klusterlet.workStore.Add(work2)
	syncWork(t, klusterlet, work2)
	for _, action := range hcmclient.Actions() {
		if action.Matches("update", "works") {
			updateCount++
		}
	}
	if updateCount != 0 {
		t.Errorf("processWork() workToUpdate = %v, want %v", updateCount, 0)
	}
}

func TestGetReleaseWork(t *testing.T) {
	work := newWork("work1", v1beta1.ResourceWorkType)
	work.ObjectMeta.Labels = map[string]string{}
	work.Spec.Scope.ResourceType = "releases"
	klusterlet, _ := newTestKlusterlet(apiConfig, &clusterv1alpha1.Cluster{})
	hcmclient := klusterlet.fakeHCMClient
	klusterlet.workStore.Add(work)
	syncWork(t, klusterlet, work)

	var updateCount int
	for _, action := range hcmclient.Actions() {
		if action.Matches("update", "works") {
			updateCount++
		}
	}

	if updateCount != 0 {
		t.Errorf("processWork() workToUpdate = %v, want %v", updateCount, 0)
	}
}

func serilizeOrDie(t *testing.T, object interface{}) []byte {
	data, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestKubeActionWork(t *testing.T) {
	//test kube create
	work := newWork("work1", v1beta1.ActionWorkType)
	work.Spec.ActionType = v1beta1.CreateActionType
	work.Spec.KubeWork = &v1beta1.KubeWorkSpec{
		Namespace: "kube-system",
		ObjectTemplate: runtime.RawExtension{
			Object: klusterletIngress,
		},
	}
	klusterlet, _ := newTestKlusterlet(apiConfig, &clusterv1alpha1.Cluster{})
	hcmclient := klusterlet.fakeHCMClient
	klusterlet.workStore.Add(work)
	syncWork(t, klusterlet, work)

	var updateCount int
	for _, action := range hcmclient.Actions() {
		if action.Matches("update", "works") {
			updateCount++
		}
	}
	if updateCount != 1 {
		t.Errorf("processWork() workToUpdate = %v, want %v", updateCount, 1)
	}

	//Test kube update
	gvk := klusterletIngress.GetObjectKind().GroupVersionKind()
	klusterlet.kubeControl.SetObject(&gvk, "", "kube-system", "mcm-ingress-testcluster-klusterlet", klusterletIngress)
	work1 := newWork("work1", v1beta1.ActionWorkType)
	work1.Spec.ActionType = v1beta1.UpdateActionType
	work1.Spec.KubeWork = &v1beta1.KubeWorkSpec{
		ObjectTemplate: runtime.RawExtension{
			Raw: serilizeOrDie(t, klusterletIngress),
		},
	}
	klusterlet.workStore.Add(work1)
	syncWork(t, klusterlet, work1)

	work2 := newWork("work2", v1beta1.ActionWorkType)
	work2.Spec.ActionType = v1beta1.UpdateActionType
	klusterletIngress2 := klusterletIngress.DeepCopy()
	klusterletIngress2.Spec.Rules = []extensionv1beta1.IngressRule{
		{
			Host: "test",
		},
	}
	work2.Spec.KubeWork = &v1beta1.KubeWorkSpec{
		ObjectTemplate: runtime.RawExtension{
			Raw: serilizeOrDie(t, klusterletIngress2),
		},
	}
	klusterlet.workStore.Add(work2)
	syncWork(t, klusterlet, work2)

	//Test kube delete
	work3 := newWork("work3", v1beta1.ActionWorkType)
	work3.Spec.ActionType = v1beta1.DeleteActionType
	work3.Spec.KubeWork = &v1beta1.KubeWorkSpec{
		Name:      "test1",
		Namespace: "test1",
		Resource:  "pods",
	}
	klusterlet.workStore.Add(work3)
	syncWork(t, klusterlet, work3)
}

func TestHelmActionWork(t *testing.T) {
	klusterlet, helmclient := newTestKlusterlet(apiConfig, &clusterv1alpha1.Cluster{})
	//Test helm create
	work1 := newWork("work1", v1beta1.ActionWorkType)
	work1.Spec.ActionType = v1beta1.CreateActionType
	work1.Spec.HelmWork = &v1beta1.HelmWorkSpec{
		ReleaseName: "mcmrelease-test2",
		ChartURL:    "http://test",
		Version:     "1.0",
	}
	klusterlet.workStore.Add(work1)
	syncWork(t, klusterlet, work1)

	if len((helmclient).Rels) != 1 {
		t.Errorf("syncDeployer() release = %v, want %v", len(helmclient.Rels), 1)
	}

	//Test helm update
	work2 := newWork("work2", v1beta1.ActionWorkType)
	work2.Spec.ActionType = v1beta1.UpdateActionType
	work2.Spec.HelmWork = &v1beta1.HelmWorkSpec{
		ReleaseName: "mcmrelease-test2",
		ChartURL:    "http://test",
		Version:     "2.0",
	}
	klusterlet.workStore.Add(work2)
	syncWork(t, klusterlet, work2)

	if len((helmclient).Rels) != 1 {
		t.Errorf("syncDeployer() release = %v, want %v", len(helmclient.Rels), 1)
	}

	//Test helm delete
	work3 := newWork("work3", v1beta1.ActionWorkType)
	work3.Spec.ActionType = v1beta1.DeleteActionType
	work3.Spec.HelmWork = &v1beta1.HelmWorkSpec{
		ReleaseName: "mcmrelease-test2",
	}
	klusterlet.workStore.Add(work3)
	syncWork(t, klusterlet, work3)

	if len((helmclient).Rels) != 0 {
		t.Errorf("syncDeployer() release = %v, want %v", len(helmclient.Rels), 0)
	}
}
