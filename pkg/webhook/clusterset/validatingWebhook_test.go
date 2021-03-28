package clusterset

import (
	v1 "k8s.io/api/admission/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"reflect"
	"testing"
)

const (
	clusterDeploymentWithLabel    = `{"apiVersion": "hive.openshift.io/v1","kind": "ClusterDeployment","metadata": {"name": "test","namespace": "default","labels": {"cluster.open-cluster-management.io/clusterset":"clusterset1"}}}`
	clusterDeploymentWithLabel2   = `{"apiVersion": "hive.openshift.io/v1","kind": "ClusterDeployment","metadata": {"name": "test","namespace": "default","labels": {"cluster.open-cluster-management.io/clusterset":"clusterset2"}}}`
	clusterDeploymentWithoutLabel = `{"apiVersion": "hive.openshift.io/v1","kind": "ClusterDeployment","metadata": {"name": "test","namespace": "default"}}`
	clusterDeploymentWithClaim    = `{"apiVersion": "hive.openshift.io/v1","kind": "ClusterDeployment","metadata": {"name": "test","namespace": "default"},"spec": {"baseDomain":"dev04.red-chesterfield.com","clusterPoolRef": {"claimName": "test"}}}`
	clusterPool                   = `{"apiVersion": "hive.openshift.io/v1","kind": "ClusterPool","metadata": {"name": "test","namespace": "default","labels": {"cluster.open-cluster-management.io/clusterset":"clusterset1"}}}`
)

func TestAdmissionHandler_ServerValidateResource(t *testing.T) {
	cases := []struct {
		name                   string
		request                *v1.AdmissionRequest
		expectedResponse       *v1.AdmissionResponse
		allowUpdateClusterSets map[string]bool
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
			name: "allow to create clusterdeployment without clusterset label",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithoutLabel),
				},
			},
			allowUpdateClusterSets: map[string]bool{"*": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "forbidden to create clusterdeployment without clusterset label",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithoutLabel),
				},
			},
			allowUpdateClusterSets: map[string]bool{"*": false},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "allow to create clusterdeployment in clusterset1",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithLabel),
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "forbidden to add clusterdeployment into clusterset1",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Update,
				OldObject: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithoutLabel),
				},
				Object: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithLabel),
				},
			},
			allowUpdateClusterSets: map[string]bool{"*": false, "clusterset1": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "allow to update clusterdeployment from clusterset1 to clusterset2",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Update,
				OldObject: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithLabel),
				},
				Object: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithLabel2),
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset1": true, "clusterset2": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "forbidden to remove clusterdeployment from clusterset1",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Update,
				OldObject: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithLabel),
				},
				Object: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithoutLabel),
				},
			},
			allowUpdateClusterSets: map[string]bool{"*": false, "clusterset1": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "forbidden to create clusterdeployment in clusterset1 with clusterset2 label",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithLabel),
				},
			},
			allowUpdateClusterSets: map[string]bool{"clusterset2": true},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "allow to create clusterdeployment by clusterclaim",
			request: &v1.AdmissionRequest{
				Resource:  clusterDeploymentsGVR,
				Operation: v1.Create,
				Object: runtime.RawExtension{
					Raw: []byte(clusterDeploymentWithClaim),
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
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kubeClient := kubefake.NewSimpleClientset()

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

			admissionHandler := &AdmissionHandler{KubeClient: kubeClient}

			actualResponse := admissionHandler.validateResource(c.request)

			if !reflect.DeepEqual(actualResponse.Allowed, c.expectedResponse.Allowed) {
				t.Errorf("expected %#v but got: %#v", c.expectedResponse, actualResponse)
			}
		})
	}
}
