package validating

import (
	"reflect"
	"testing"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hivefake "github.com/openshift/hive/pkg/client/clientset/versioned/fake"
	v1 "k8s.io/api/admission/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

const (
	clusterPool                             = `{"apiVersion": "hive.openshift.io/v1","kind": "ClusterPool","metadata": {"name": "test","namespace": "default","labels": {"cluster.open-cluster-management.io/clusterset":"clusterset1"}}}`
	updateClusterPoolSet                    = `{"apiVersion": "hive.openshift.io/v1","kind": "ClusterPool","metadata": {"name": "test","namespace": "default","labels": {"cluster.open-cluster-management.io/clusterset":"clusterset2"}}}`
	updateClusterPoolNotSet                 = `{"apiVersion":"hive.openshift.io/v1","kind":"ClusterPool","metadata":{"name":"test","namespace":"default","labels":{"cluster.open-cluster-management.io/clusterset":"clusterset1","a":"b"}}}`
	managedcluster                          = `{"apiVersion":"cluster.open-cluster-management.io/v1","kind":"ManagedCluster","metadata":{"labels":{"cluster.open-cluster-management.io/clusterset":"clusterset1"},"name":"c0"},"spec":{}}`
	updateManagedclusterSet                 = `{"apiVersion":"cluster.open-cluster-management.io/v1","kind":"ManagedCluster","metadata":{"labels":{"cluster.open-cluster-management.io/clusterset":"clusterset2"},"name":"c0"},"spec":{}}`
	updateManagedclusterNotSet              = `{"apiVersion":"cluster.open-cluster-management.io/v1","kind":"ManagedCluster","metadata":{"labels":{"cluster.open-cluster-management.io/clusterset":"clusterset1","a":"b"},"name":"c0"},"spec":{}}`
	managedclusterDefaultSet                = `{"apiVersion":"cluster.open-cluster-management.io/v1","kind":"ManagedCluster","metadata":{"labels":{"cluster.open-cluster-management.io/clusterset":"default"},"name":"c0"},"spec":{}}`
	managedclusterNoSet                     = `{"apiVersion":"cluster.open-cluster-management.io/v1","kind":"ManagedCluster","metadata":{"name":"c0"},"spec":{}}`
	createClusterDeploymentFromPool         = `{"apiVersion":"hive.openshift.io/v1","kind":"ClusterDeployment","metadata":{"name":"cd-pool","namespace":"cd-pool"},"spec":{"clusterName":"gcp-pool-9m688","clusterPoolRef":{"namespace":"pool","poolName":"gcp-pool"}}}`
	createClusterDeploymentInSet            = `{"apiVersion":"hive.openshift.io/v1","kind":"ClusterDeployment","metadata":{"name":"cd-pool","namespace":"cd-pool","labels":{"cluster.open-cluster-management.io/clusterset":"clusterset1"}},"spec":{"clusterName":"gcp-pool-9m688"}}`
	createClusterDeploymentFromAgentCluster = `{"apiVersion":"hive.openshift.io/v1","kind":"ClusterDeployment","metadata":{"name":"cd-pool","namespace":"cd-pool","ownerReferences":[{"apiVersion":"capi-provider.agent-install.openshift.io/v1","kind":"AgentCluster"}]},"spec":{"clusterName":"gcp-pool-9m688"}}`
	labelSelectorSet                        = `{"apiVersion":"cluster.open-cluster-management.io/v1beta2","kind":"ManagedClusterSet","metadata":{"name":"te-label-set"},"spec":{"clusterSelector":{"labelSelector":{"matchLabels":{"vendor":"ocp"}},"selectorType":"LabelSelector"}}}`
	globalSet                               = `{"apiVersion":"cluster.open-cluster-management.io/v1beta2","kind":"ManagedClusterSet","metadata":{"name":"global"},"spec":{"clusterSelector":{"labelSelector":{},"selectorType":"LabelSelector"}}}`
	defaultSet                              = `{"apiVersion":"cluster.open-cluster-management.io/v1beta2","kind":"ManagedClusterSet","metadata":{"name":"default"},"spec":{"clusterSelector":{"selectorType":"ExclusiveClusterSetLabel"}}}`
	errorDefaultSet                         = `{"apiVersion":"cluster.open-cluster-management.io/v1beta2","kind":"ManagedClusterSet","metadata":{name":"default"},"spec":{"clusterSelector":{"selectorType":"ExclusiveClusterSetLabel"}}}`
)

func TestAdmissionHandler_ServerValidateResource(t *testing.T) {
	cases := []struct {
		name                      string
		request                   *v1.AdmissionRequest
		existingClusterdeployment []runtime.Object
		expectedResponse          *v1.AdmissionResponse
		allowUpdateClusterSets    map[string]bool
	}{
		{
			name: "validate none specified resources request",
			request: &v1.AdmissionRequest{
				Resource: metav1.GroupVersionResource{
					Group:    "test.open-cluster-management.io",
					Version:  "v1",
					Resource: "tests",
				},
			},

			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to create clusterpool in clusterset1",
			request: &v1.AdmissionRequest{
				Resource:  clusterPoolsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(clusterPool),
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "deny to update clusterpool from clusterset1 in clusterset2",
			request: &v1.AdmissionRequest{
				Resource:  clusterPoolsGVR,
				Operation: v1.Update,
				Object: runtime.RawExtension{
					Raw: []byte(updateClusterPoolSet),
				},
				OldObject: runtime.RawExtension{
					Raw: []byte(clusterPool),
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "allow to update clusterpool if not updating set",
			request: &v1.AdmissionRequest{
				Resource:  clusterPoolsGVR,
				Operation: v1.Update,
				Object: runtime.RawExtension{
					Raw: []byte(updateClusterPoolNotSet),
				},
				OldObject: runtime.RawExtension{
					Raw: []byte(clusterPool),
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to create cluster in clusterset1",
			request: &v1.AdmissionRequest{
				Resource:  clusterPoolsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(managedcluster),
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to update cluster with clusterset label unchanged",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Update,
				Object: runtime.RawExtension{
					Raw: []byte(updateManagedclusterNotSet),
				},
				OldObject: runtime.RawExtension{
					Raw: []byte(managedcluster),
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to update cluster to clusterset2 if there is no clusterdeployment",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Update,
				Object: runtime.RawExtension{
					Raw: []byte(updateManagedclusterSet),
				},
				OldObject: runtime.RawExtension{
					Raw: []byte(managedcluster),
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true, "clusterset2": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to update cluster to clusterset2 if there is a clusterdeployment",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Update,
				Object: runtime.RawExtension{
					Raw: []byte(updateManagedclusterSet),
				},
				OldObject: runtime.RawExtension{
					Raw: []byte(managedcluster),
				},
			},
			existingClusterdeployment: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "c0",
						Namespace: "c0",
					},
					Spec: hivev1.ClusterDeploymentSpec{},
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true, "clusterset2": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "deny to update cluster to clusterset2 if there is a clusterdeployment",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Update,
				Object: runtime.RawExtension{
					Raw: []byte(updateManagedclusterSet),
				},
				OldObject: runtime.RawExtension{
					Raw: []byte(managedcluster),
				},
			},
			existingClusterdeployment: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "c0",
						Namespace: "c0",
					},
					Spec: hivev1.ClusterDeploymentSpec{
						ClusterPoolRef: &hivev1.ClusterPoolReference{
							PoolName:  "p1",
							Namespace: "ns1",
						},
					},
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true, "clusterset2": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "allow to update cluster to default if there is a clusterdeployment from pool",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Update,
				Object: runtime.RawExtension{
					Raw: []byte(managedclusterDefaultSet),
				},
				OldObject: runtime.RawExtension{
					Raw: []byte(managedclusterNoSet),
				},
			},
			existingClusterdeployment: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "c0",
						Namespace: "c0",
					},
					Spec: hivev1.ClusterDeploymentSpec{
						ClusterPoolRef: &hivev1.ClusterPoolReference{
							PoolName:  "p1",
							Namespace: "ns1",
						},
					},
				},
			},
			allowUpdateClusterSets: map[string]bool{"*": true, "default": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow create clusterdeployment if from pool",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(createClusterDeploymentFromPool),
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "deny create clusterdeployment if has set",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(createClusterDeploymentInSet),
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "allow create clusterdeployment from agentcluster",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(createClusterDeploymentFromAgentCluster),
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow create default clusterset",
			request: &v1.AdmissionRequest{
				Resource:  managedClusterSetsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(defaultSet),
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow create global clusterset",
			request: &v1.AdmissionRequest{
				Resource:  managedClusterSetsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(globalSet),
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow delete global clusterset",
			request: &v1.AdmissionRequest{
				Resource:  managedClusterSetsGVR,
				Operation: v1.Delete,
				Object: runtime.RawExtension{
					Raw: []byte(globalSet),
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "do not allow create label selector clusterset",
			request: &v1.AdmissionRequest{
				Resource:  managedClusterSetsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(labelSelectorSet),
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "request error for clusterset",
			request: &v1.AdmissionRequest{
				Resource:  managedClusterSetsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(errorDefaultSet),
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kubeClient := kubefake.NewSimpleClientset()
			hiveClient := hivefake.NewSimpleClientset(c.existingClusterdeployment...)
			kubeClient.PrependReactor(
				"create",
				"subjectaccessreviews",
				func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
					allowed := false

					sar := action.(clienttesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview)
					switch sar.Spec.ResourceAttributes.Resource {
					case "managedclustersets":
						if sar.Spec.ResourceAttributes.Name == "" {
							allowed = c.allowUpdateClusterSets["*"]
						} else {
							allowed = c.allowUpdateClusterSets[sar.Spec.ResourceAttributes.Name]
						}
					}

					return true, &authorizationv1.SubjectAccessReview{
						Status: authorizationv1.SubjectAccessReviewStatus{
							Allowed: allowed,
						},
					}, nil
				},
			)

			admissionHandler := &AdmissionHandler{KubeClient: kubeClient, HiveClient: hiveClient}

			actualResponse := admissionHandler.validateResource(c.request)

			if !reflect.DeepEqual(actualResponse.Allowed, c.expectedResponse.Allowed) {
				t.Errorf("case: %v,expected %#v but got: %#v", c.name, c.expectedResponse, actualResponse)
			}
		})
	}
}
