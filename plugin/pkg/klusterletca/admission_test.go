// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterletca

import (
	"context"
	"os"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/admission"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	hadmission "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apiserver/admission"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/internalclientset"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/internalclientset/fake"
	informers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/internalversion"
	"k8s.io/apiserver/pkg/authentication/user"
	certutil "k8s.io/client-go/util/cert"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// newHandlerForTest returns a configured handler for testing.
func newHandlerForTest(internalClient internalclientset.Interface, caFile *string) (admission.Interface, informers.SharedInformerFactory, error) {
	f := informers.NewSharedInformerFactory(internalClient, 5*time.Minute)
	handler, err := NewKlusterletCAAppend(caFile)
	if err != nil {
		return nil, f, err
	}
	pluginInitializer := hadmission.NewPluginInitializer(internalClient, f, nil, nil, nil, nil)
	pluginInitializer.Initialize(handler)
	err = admission.ValidateInitialization(handler)
	return handler, f, err
}

// newFakeHCMClientForTest creates a fake clientset that returns a
// worklist with the given work as the single list item.
func newFakeHCMClientForTest(objects ...runtime.Object) *fake.Clientset {
	return fake.NewSimpleClientset(objects...)
}

// newWork returns a new work for the specified namespace.
func newClusterStatus(namespace string, name string) *mcm.ClusterStatus {
	clusterStatus := &mcm.ClusterStatus{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       mcm.ClusterStatusSpec{},
	}
	return clusterStatus
}

func newCluster() *clusterv1alpha1.Cluster {
	return &clusterv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: "test",
		},
	}
}

// TestClusterServicePlanChangeBlockedByUpdateablePlanSetting tests that the
// Admission Controller will block a request to update an Instance's
// Service Plan
func TestClusterStatusAppend(t *testing.T) {
	cert, _, err := certutil.GenerateSelfSignedCertKey("localhost", nil, nil)
	if err != nil {
		t.Errorf("Failed to generate certificae")
	}

	certFile := "/tmp/test.crt"
	err = certutil.WriteCert(certFile, cert)
	if err != nil {
		t.Errorf("failed to write cert")
	}
	defer os.Remove(certFile)

	clusterStatus := newClusterStatus("cluster1", "cluster1")
	fakeClient := newFakeHCMClientForTest(clusterStatus)
	handler, informerFactory, err := newHandlerForTest(fakeClient, &certFile)
	if err != nil {
		t.Errorf("unexpected error initializing handler: %v", err)
	}
	informerFactory.Start(wait.NeverStop)
	cluster := newCluster()
	err = handler.(admission.MutationInterface).Admit(
		context.TODO(),
		admission.NewAttributesRecord(
			cluster,
			nil,
			clusterv1alpha1.Kind("cluster").WithVersion("version"),
			cluster.Namespace,
			cluster.Name,
			clusterv1alpha1.Resource("clusters").WithVersion("version"),
			"",
			admission.Create,
			&metav1.CreateOptions{},
			true,
			&user.DefaultInfo{},
		),
		nil,
	)
	if err != nil {
		t.Errorf("unexpected error %q returned from admission handler.", err.Error())
	}
}
