package autodetect

import (
	"context"
	stdlog "log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/onsi/gomega"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cliScheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	c      client.Client
	cfg    *rest.Config
	scheme = runtime.NewScheme()
)

const (
	ManagedClusterName = "foo"
)

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deploy", "foundation", "hub", "resources", "crds")},
	}

	clusterv1.AddToScheme(cliScheme.Scheme)
	clusterv1beta1.AddToScheme(cliScheme.Scheme)

	var err error
	if cfg, err = t.Start(); err != nil {
		stdlog.Fatal(err)
	}

	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, clusterv1.Install, clusterv1beta1.AddToScheme)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := clusterv1beta1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding cluster info to scheme, %v", err)
		os.Exit(1)
	}
	if err := clusterv1.Install(scheme); err != nil {
		klog.Errorf("Failed adding cluster to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

// StartTestManager adds recFn
func StartTestManager(mgr manager.Manager, g *gomega.GomegaWithT) (chan struct{}, *sync.WaitGroup) {
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		g.Expect(mgr.Start(stop)).NotTo(gomega.HaveOccurred())
	}()

	return stop, wg
}

func TestControllerReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	SetupWithManager(mgr)

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	time.Sleep(time.Second * 1)
}

func validateError(t *testing.T, err, expectedErrorType error) {
	if expectedErrorType != nil {
		assert.EqualError(t, err, expectedErrorType.Error())
	} else {
		assert.NoError(t, err)
	}
}

func newTestReconciler(existingObjs []runtime.Object) (*Reconciler, client.Client) {
	client := fake.NewFakeClientWithScheme(scheme, existingObjs...)
	return &Reconciler{
		client: client,
		scheme: scheme,
	}, client
}

func TestReconcile(t *testing.T) {
	tests := []struct {
		name              string
		existingObjs      []runtime.Object
		expectedErrorType error
		req               reconcile.Request
		requeue           bool
	}{
		{
			name:         "ManagedClusterNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
			expectedErrorType: nil,
			requeue:           false,
		},
		{
			name: "ManagedClusterInfoNotFound",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
						Labels: map[string]string{
							LabelCloudVendor: AutoDetect,
							LabelKubeVendor:  AutoDetect,
						},
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
			expectedErrorType: nil,
			requeue:           true,
		},
		{
			name: "UpdateManagedClusterLabels",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
						Labels: map[string]string{
							LabelCloudVendor: AutoDetect,
							LabelKubeVendor:  AutoDetect,
						},
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedClusterName,
						Namespace: ManagedClusterName,
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						KubeVendor:  clusterv1beta1.KubeVendorAKS,
						CloudVendor: clusterv1beta1.CloudVendorAzure,
						ClusterID:   "c186d39e-f56f-45c3-8869-fc84323165c4",
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
			expectedErrorType: nil,
			requeue:           false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svrc, _ := newTestReconciler(test.existingObjs)
			res, err := svrc.Reconcile(test.req)
			validateError(t, err, test.expectedErrorType)
			if test.requeue {
				assert.Equal(t, res.Requeue, true)
			} else {
				assert.Equal(t, res.Requeue, false)
			}
		})
	}
}

func TestOSDVendor(t *testing.T) {
	tests := []struct {
		name              string
		existingObjs      []runtime.Object
		expectedErrorType error
		req               reconcile.Request
		expectedLabel     map[string]string
	}{
		{
			name: "UpdateManagedClusterLabelsOpenShift",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
						Labels: map[string]string{
							LabelCloudVendor: AutoDetect,
							LabelKubeVendor:  AutoDetect,
						},
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedClusterName,
						Namespace: ManagedClusterName,
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						KubeVendor:  clusterv1beta1.KubeVendorOpenShift,
						CloudVendor: clusterv1beta1.CloudVendorAzure,
					},
				},
			},
			expectedLabel:     map[string]string{LabelCloudVendor: "Azure", LabelKubeVendor: "OpenShift"},
			expectedErrorType: nil,
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
		},
		{
			name: "UpdateManagedClusterLabelsOpenShiftDedicated",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
						Labels: map[string]string{
							LabelCloudVendor: AutoDetect,
							LabelKubeVendor:  AutoDetect,
						},
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedClusterName,
						Namespace: ManagedClusterName,
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						KubeVendor:  clusterv1beta1.KubeVendorOSD,
						CloudVendor: clusterv1beta1.CloudVendorAzure,
					},
				},
			},
			expectedLabel:     map[string]string{LabelCloudVendor: "Azure", LabelKubeVendor: "OpenShift", LabelManagedBy: "platform"},
			expectedErrorType: nil,
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svrc, client := newTestReconciler(test.existingObjs)
			_, err := svrc.Reconcile(test.req)
			validateError(t, err, test.expectedErrorType)
			cluster := &clusterv1.ManagedCluster{}
			err = client.Get(context.Background(), types.NamespacedName{Name: ManagedClusterName}, cluster)
			validateError(t, err, nil)
			if !reflect.DeepEqual(cluster.Labels, test.expectedLabel) {
				t.Errorf("Labels not equal, actual %v, expected %v", cluster.Labels, test.expectedLabel)
			}
		})
	}
}
