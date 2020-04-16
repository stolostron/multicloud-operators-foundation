// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package bootstrap

import (
	"testing"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	hcmv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	hcmfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	clusterfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset/fake"
	clusterinformers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	informers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common"
	csrv1beta1 "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func newController(cjr *v1alpha1.ClusterJoinRequest, csr *csrv1beta1.CertificateSigningRequest) *Controller {
	KubeClient := kubefake.NewSimpleClientset(csr)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(KubeClient, time.Minute*10)

	hcmClient := hcmfake.NewSimpleClientset(cjr)
	hcmInformerFactory := informers.NewSharedInformerFactory(hcmClient, time.Minute*10)

	clusterClient := clusterfake.NewSimpleClientset()
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterClient, time.Minute*10)

	return NewController(KubeClient, hcmClient, kubeInformerFactory, hcmInformerFactory, clusterInformerFactory, true, nil)
}

func newCJR(namespace, name string, renewal bool) *v1alpha1.ClusterJoinRequest {
	cjr := &v1alpha1.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cjr1",
		},
		Spec: v1alpha1.ClusterJoinRequestSpec{
			ClusterNamespace: namespace,
			ClusterName:      name,
		},
	}

	if renewal {
		cjr.Annotations = map[string]string{
			common.RenewalAnnotation: "true",
		}
	}

	return cjr
}

func newCSR(namespace, name string) *csrv1beta1.CertificateSigningRequest {
	return &csrv1beta1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newClusters(clusterMeta []metav1.ObjectMeta) []*clusterv1alpha1.Cluster {
	clusters := make([]*clusterv1alpha1.Cluster, 0)
	for _, m := range clusterMeta {
		clusters = append(clusters, &clusterv1alpha1.Cluster{
			ObjectMeta: m,
			Status: clusterv1alpha1.ClusterStatus{
				Conditions: []clusterv1alpha1.ClusterCondition{
					{
						Type: clusterv1alpha1.ClusterOK,
					},
				},
			},
		})
	}

	return clusters
}

func TestApproveJoinCJR(t *testing.T) {
	hcmjoin := newCJR("c1", "c1", false)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)
	clusters := newClusters(nil)

	controller := newController(hcmjoin, csr)
	err := controller.approveOrDenyClusterJoinRequest(hcmjoin, csr, false, clusters)
	if err != nil {
		t.Errorf("error to approve cjr: %v", err)
	}

	if len(csr.Status.Conditions) == 0 {
		t.Errorf("CSR should be approved")
	} else if csr.Status.Conditions[0].Type != csrv1beta1.CertificateApproved {
		t.Errorf("Wrong condition type: %v", csr.Status.Conditions[0].Type)
	}
}

func TestApproveJoinCJRFromOfflineCluster(t *testing.T) {
	hcmjoin := newCJR("c1", "c1", false)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)
	clusters := []*clusterv1alpha1.Cluster{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "c1",
				Name:      "c1",
			},
		},
	}

	controller := newController(hcmjoin, csr)
	err := controller.approveOrDenyClusterJoinRequest(hcmjoin, csr, false, clusters)
	if err != nil {
		t.Errorf("error to approve cjr: %v", err)
	}

	if len(csr.Status.Conditions) == 0 {
		t.Errorf("CSR should be approved")
	} else if csr.Status.Conditions[0].Type != csrv1beta1.CertificateApproved {
		t.Errorf("Wrong condition type: %v", csr.Status.Conditions[0].Type)
	}
}

func TestDenyJoinCJRWithDuplicatedNamespace(t *testing.T) {
	hcmjoin := newCJR("c1", "c1", false)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)
	clusters := newClusters([]metav1.ObjectMeta{
		{
			Namespace: "c1",
			Name:      "c2",
		},
	})

	controller := newController(hcmjoin, csr)
	err := controller.approveOrDenyClusterJoinRequest(hcmjoin, csr, false, clusters)
	if err != nil {
		t.Errorf("error to approve cjr: %v", err)
	}

	if hcmjoin.Status.Phase != hcmv1alpha1.JoinDenied {
		t.Errorf("CJR should be denied for cluster namespace exists: c1")
	}
}

func TestDenyJoinCJRWithDuplicatedName(t *testing.T) {
	hcmjoin := newCJR("c1", "c1", false)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)
	clusters := newClusters([]metav1.ObjectMeta{
		{
			Namespace: "c2",
			Name:      "c1",
		},
	})

	controller := newController(hcmjoin, csr)
	err := controller.approveOrDenyClusterJoinRequest(hcmjoin, csr, false, clusters)
	if err != nil {
		t.Errorf("error to approve cjr: %v", err)
	}

	if hcmjoin.Status.Phase != hcmv1alpha1.JoinDenied {
		t.Errorf("CJR should be denied for cluster name exists: c1")
	}
}

func TestApproveRenewalCJR(t *testing.T) {
	hcmjoin := newCJR("c1", "c1", true)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)
	clusters := newClusters([]metav1.ObjectMeta{
		{
			Namespace: "c1",
			Name:      "c1",
		},
	})

	controller := newController(hcmjoin, csr)
	err := controller.approveOrDenyClusterJoinRequest(hcmjoin, csr, false, clusters)
	if err != nil {
		t.Errorf("error to approve cjr: %v", err)
	}

	if len(csr.Status.Conditions) == 0 {
		t.Errorf("CSR is not approved")
	} else if csr.Status.Conditions[0].Type != csrv1beta1.CertificateApproved {
		t.Errorf("Wrong condition type: %v", csr.Status.Conditions[0].Type)
	}
}

func TestDenyRenewalCJRFromUnknownCluster(t *testing.T) {
	hcmjoin := newCJR("c2", "c2", true)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)
	clusters := newClusters([]metav1.ObjectMeta{
		{
			Namespace: "c2",
			Name:      "c1",
		},
	})

	controller := newController(hcmjoin, csr)
	err := controller.approveOrDenyClusterJoinRequest(hcmjoin, csr, false, clusters)
	if err != nil {
		t.Errorf("error to approve cjr: %v", err)
	}

	if hcmjoin.Status.Phase != hcmv1alpha1.JoinDenied {
		t.Errorf("CJR should be denied for cluster does not exists")
	}
}

func TestIgnoreRenewalCJRFromOfflineCluster(t *testing.T) {
	hcmjoin := newCJR("c2", "c2", true)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)
	clusters := []*clusterv1alpha1.Cluster{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "c2",
				Name:      "c2",
			},
		},
	}

	controller := newController(hcmjoin, csr)
	err := controller.approveOrDenyClusterJoinRequest(hcmjoin, csr, false, clusters)
	if err != nil {
		t.Errorf("error to approve cjr: %v", err)
	}

	if hcmjoin.Status.Phase != "" {
		t.Errorf("CJR should not be approved or denied for cluster is pending or offline")
	}

	if len(csr.Status.Conditions) > 0 {
		t.Errorf("CSR should not be approved or denied for cluster is pending or offline: \n%+v", csr.Status)
	}
}

func TestCreateRoles(t *testing.T) {
	hcmjoin := newCJR("c2", "c2", true)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)

	controller := newController(hcmjoin, csr)
	hcmJoinReq := &hcmv1alpha1.ClusterJoinRequest{
		Spec: hcmv1alpha1.ClusterJoinRequestSpec{
			ClusterNamespace: "c2",
		},
	}
	err := controller.createRoles(hcmJoinReq)
	if err != nil {
		t.Errorf("fail to create roles")
	}
}

func TestHandle(t *testing.T) {
	hcmjoin := newCJR("c2", "c2", true)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)

	controller := newController(hcmjoin, csr)
	controller.hcmJoinHandler("c2/c2")
	controller.csrHandler("c2/c2")
}

func TestProcessNextWorkItem(t *testing.T) {
	hcmjoin := newCJR("c2", "c2", true)
	csr := newCSR(hcmjoin.Namespace, hcmjoin.Name)

	controller := newController(hcmjoin, csr)
	controller.enqueue(csr, controller.csrworkqueue)
	controller.enqueue(hcmjoin, controller.hcmjoinworkqueue)
	controller.processNextWorkItem(controller.csrworkqueue, controller.csrHandler)
	controller.processNextWorkItem(controller.hcmjoinworkqueue, controller.hcmJoinHandler)
}
