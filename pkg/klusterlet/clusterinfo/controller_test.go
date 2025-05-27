package clusterinfo

import (
	"context"

	apiconfigv1 "github.com/openshift/api/config/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	clusterv1 "open-cluster-management.io/api/cluster/v1"

	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	routefake "github.com/openshift/client-go/route/clientset/versioned/fake"
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var clusterName = "c1"
var clusterc1Request = reconcile.Request{
	NamespacedName: types.NamespacedName{
		Name: clusterName, Namespace: clusterName}}

var existingKubeObjs = []runtime.Object{
	newDeployment("klusterlet-addon-workmgr", "myns"),
	&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-token-xxx",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"ca.crt": []byte("sa-fake-cert-data"),
		},
		Type: corev1.SecretTypeServiceAccountToken,
	},
	&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "kube-system",
		},
		Secrets: []corev1.ObjectReference{
			{
				Name: "default-token-xxx",
			},
		},
	},
}

var existingOcpObjs = []runtime.Object{
	newClusterVersion(),
	&apiconfigv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: apiconfigv1.InfrastructureSpec{},
		Status: apiconfigv1.InfrastructureStatus{
			APIServerURL: "http://127.0.0.1:6443",
		},
	},
}

var claims = []runtime.Object{
	newClaim(clusterclaim.ClaimOCMConsoleURL, "https://abc.com"),
	newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
	newClaim(clusterclaim.ClaimOCMProduct, clusterclaim.ProductOpenShift),
	newClaim(clusterclaim.ClaimOCMPlatform, clusterclaim.PlatformAWS),
	newClaim(clusterclaim.ClaimOpenshiftID, "aaa-bbb"),
}

func NewClusterInfoReconciler() *ClusterInfoReconciler {
	fakeKubeClient := kubefake.NewSimpleClientset(existingKubeObjs...)
	fakeRouteV1Client := routefake.NewSimpleClientset()
	fakeClusterClient := clusterfake.NewSimpleClientset(claims...)
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
	clusterStore := clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Informer().GetStore()
	for _, item := range claims {
		clusterStore.Add(item)
	}

	return &ClusterInfoReconciler{
		Log:                     ctrl.Log.WithName("controllers").WithName("ManagedClusterInfo"),
		ManagementClusterClient: fakeKubeClient,
		ManagedClusterClient:    fakeKubeClient,
		ClusterName:             clusterName,
		ClaimLister:             clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Lister(),
		RouteV1Client:           fakeRouteV1Client,
		ConfigV1Client:          configfake.NewSimpleClientset(existingOcpObjs...),
	}
}

func NewFailedClusterInfoReconciler() *ClusterInfoReconciler {
	fakeKubeClient := kubefake.NewSimpleClientset(newMonitoringSecret())
	fakeRouteV1Client := routefake.NewSimpleClientset()
	fakeClusterClient := clusterfake.NewSimpleClientset()
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
	return &ClusterInfoReconciler{
		Log:                     ctrl.Log.WithName("controllers").WithName("ManagedClusterInfo"),
		ManagementClusterClient: fakeKubeClient,
		ManagedClusterClient:    fakeKubeClient,
		ClusterName:             clusterName,
		ClaimLister:             clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Lister(),
		RouteV1Client:           fakeRouteV1Client,
	}
}

func TestClusterInfoReconcile(t *testing.T) {
	ctx := context.Background()
	// Create new cluster
	now := metav1.Now()
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:              clusterName,
			Namespace:         clusterName,
			CreationTimestamp: now,
		},
		Status: clusterv1beta1.ClusterInfoStatus{
			KubeVendor: clusterv1beta1.KubeVendorOpenShift,
			Conditions: []metav1.Condition{
				{
					Type:   clusterv1.ManagedClusterConditionAvailable,
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(clusterv1beta1.SchemeGroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	clusterv1beta1.AddToScheme(s)

	c := fake.NewClientBuilder().WithObjects(clusterInfo).WithStatusSubresource(clusterInfo).WithScheme(s).Build()

	fr := NewClusterInfoReconciler()

	fr.Client = c

	_, err := fr.Reconcile(ctx, clusterc1Request)
	if err != nil {
		t.Errorf("Failed to run reconcile cluster. error: %v", err)
	}

	updatedClusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err = fr.Get(context.Background(), clusterc1Request.NamespacedName, updatedClusterInfo)
	if err != nil {
		t.Errorf("failed get updated clusterinfo ")
	}

	assert.Equal(t, updatedClusterInfo.Status.Version, "v1.20.0")
	assert.Equal(t, updatedClusterInfo.Status.ConsoleURL, "https://abc.com")

	if meta.IsStatusConditionFalse(updatedClusterInfo.Status.Conditions, clusterv1beta1.ManagedClusterInfoSynced) {
		t.Errorf("failed to update synced condtion")
	}
}

func TestFailedClusterInfoReconcile(t *testing.T) {
	ctx := context.Background()
	// Create new cluster
	now := metav1.Now()
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:              clusterName,
			Namespace:         clusterName,
			CreationTimestamp: now,
		},
		Status: clusterv1beta1.ClusterInfoStatus{
			KubeVendor: clusterv1beta1.KubeVendorAKS,
			Conditions: []metav1.Condition{
				{
					Type:   clusterv1.ManagedClusterConditionAvailable,
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(clusterv1beta1.SchemeGroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	clusterv1beta1.AddToScheme(s)

	c := fake.NewClientBuilder().WithScheme(s).WithObjects(clusterInfo).WithStatusSubresource(clusterInfo).Build()

	fr := NewFailedClusterInfoReconciler()

	fr.Client = c

	_, err := fr.Reconcile(ctx, clusterc1Request)
	if err != nil {
		t.Errorf("Failed to run reconcile cluster. error: %v", err)
	}

	updatedClusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err = fr.Get(context.Background(), clusterc1Request.NamespacedName, updatedClusterInfo)
	if err != nil {
		t.Errorf("failed get updated clusterinfo ")
	}

	if !meta.IsStatusConditionTrue(updatedClusterInfo.Status.Conditions, clusterv1beta1.ManagedClusterInfoSynced) {
		t.Errorf("failed to update synced condtion")
	}
}

func newDeployment(name, namespace string) *v1.Deployment {
	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
