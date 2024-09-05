package validating

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	apiconstants "github.com/stolostron/cluster-lifecycle-api/constants"
	v1 "k8s.io/api/admission/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stolostron/multicloud-operators-foundation/pkg/constants"
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

var (
	scheme = runtime.NewScheme()
)

func TestMain(m *testing.M) {
	hivev1.AddToScheme(scheme)
	m.Run()
}

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
						Name: "c0",
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
			hiveClient := fake.NewClientBuilder().WithRuntimeObjects(c.existingClusterdeployment...).WithScheme(scheme).Build()
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

			actualResponse := admissionHandler.ValidateResource(c.request)

			if !reflect.DeepEqual(actualResponse.Allowed, c.expectedResponse.Allowed) {
				t.Errorf("case: %v,expected %#v but got: %#v", c.name, c.expectedResponse, actualResponse)
			}
		})
	}
}

func TestDeleteHosteingCluster(t *testing.T) {
	cases := []struct {
		name                    string
		request                 *v1.AdmissionRequest
		existingManagedClusters []runtime.Object
		expectedResponse        *v1.AdmissionResponse
	}{
		{
			name: "allow to delete a non-hosting cluster",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "not allowed to delete a hosting cluster",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
					},
				},
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c1",
						Annotations: map[string]string{
							apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
							apiconstants.AnnotationKlusterletHostingClusterName: "c0",
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "not allowed to delete a hosting cluster with claim",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
						Labels: map[string]string{
							constants.LabelFeatureHypershiftAddon: "available",
						},
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []metav1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: metav1.ConditionTrue,
								Reason: "available",
							},
						},
						ClusterClaims: []clusterv1.ManagedClusterClaim{
							{
								Name:  constants.ClusterClaimHostedClusterCountZero,
								Value: "false",
							},
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "allow to delete a hosting cluster, cluster already in deletion status",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "c0",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []metav1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: metav1.ConditionTrue,
								Reason: "available",
							},
						},
						ClusterClaims: []clusterv1.ManagedClusterClaim{
							{
								Name:  constants.ClusterClaimHostedClusterCountZero,
								Value: "false",
							},
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to delete a hosting cluster, no claim",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
						Labels: map[string]string{
							constants.LabelFeatureHypershiftAddon: "available",
						},
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []metav1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: metav1.ConditionTrue,
								Reason: "available",
							},
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to delete a hosting cluster, cluster not available",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
						Labels: map[string]string{
							constants.LabelFeatureHypershiftAddon: "available",
						},
					},
					Status: clusterv1.ManagedClusterStatus{
						ClusterClaims: []clusterv1.ManagedClusterClaim{
							{
								Name:  constants.ClusterClaimHostedClusterCountZero,
								Value: "false",
							},
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to delete a hosting cluster, hypershift addon not available",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []metav1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: metav1.ConditionTrue,
								Reason: "available",
							},
						},
						ClusterClaims: []clusterv1.ManagedClusterClaim{
							{
								Name:  constants.ClusterClaimHostedClusterCountZero,
								Value: "false",
							},
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kubeClient := kubefake.NewSimpleClientset()
			hiveClient := fake.NewClientBuilder().WithRuntimeObjects().WithScheme(scheme).Build()
			clusterClient := clusterfake.NewSimpleClientset(c.existingManagedClusters...)
			clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
			clusterStore := clusterInformerFactory.Cluster().V1().ManagedClusters().Informer().GetStore()
			for _, cluster := range c.existingManagedClusters {
				if err := clusterStore.Add(cluster); err != nil {
					t.Fatal(err)
				}
			}

			admissionHandler := &AdmissionHandler{
				KubeClient:    kubeClient,
				HiveClient:    hiveClient,
				ClusterLister: clusterInformerFactory.Cluster().V1().ManagedClusters().Lister(),
			}

			actualResponse := admissionHandler.ValidateResource(c.request)

			if !reflect.DeepEqual(actualResponse.Allowed, c.expectedResponse.Allowed) {
				t.Errorf("case: %v,expected %#v but got: %#v", c.name, c.expectedResponse, actualResponse)
			}
		})
	}
}

func TestDeleteClusterDeployment(t *testing.T) {
	cases := []struct {
		name                       string
		request                    *v1.AdmissionRequest
		existingManagedClusters    []runtime.Object
		existingClusterDeployments []runtime.Object
		expectedResponse           *v1.AdmissionResponse
	}{
		{
			name: "not allowed to delete the clusterDeployment",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingClusterDeployments: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "c0",
						Namespace: "test",
					},
				},
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
						Labels: map[string]string{
							constants.LabelFeatureHypershiftAddon: "available",
						},
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []metav1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: metav1.ConditionTrue,
								Reason: "available",
							},
						},
						ClusterClaims: []clusterv1.ManagedClusterClaim{
							{
								Name:  constants.ClusterClaimHostedClusterCountZero,
								Value: "false",
							},
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			name: "allow to delete, managed cluster not found",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingClusterDeployments: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "c0",
						Namespace: "test",
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to delete, no claim",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingClusterDeployments: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "c0",
						Namespace: "test",
					},
				},
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
						Labels: map[string]string{
							constants.LabelFeatureHypershiftAddon: "available",
						},
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []metav1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: metav1.ConditionTrue,
								Reason: "available",
							},
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to delete, managed cluster unavailable",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingClusterDeployments: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "c0",
						Namespace: "test",
					},
				},
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
						Labels: map[string]string{
							constants.LabelFeatureHypershiftAddon: "available",
						},
					},
					Status: clusterv1.ManagedClusterStatus{
						ClusterClaims: []clusterv1.ManagedClusterClaim{
							{
								Name:  constants.ClusterClaimHostedClusterCountZero,
								Value: "false",
							},
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			name: "allow to delete, hypershift addon unavailable",
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Delete,
				Name:      "c0",
			},
			existingClusterDeployments: []runtime.Object{
				&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "c0",
						Namespace: "test",
					},
				},
			},
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []metav1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: metav1.ConditionTrue,
								Reason: "available",
							},
						},
						ClusterClaims: []clusterv1.ManagedClusterClaim{
							{
								Name:  constants.ClusterClaimHostedClusterCountZero,
								Value: "false",
							},
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kubeClient := kubefake.NewSimpleClientset()
			hiveClient := fake.NewClientBuilder().WithRuntimeObjects(c.existingClusterDeployments...).WithScheme(scheme).Build()
			clusterClient := clusterfake.NewSimpleClientset(c.existingManagedClusters...)
			clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
			clusterStore := clusterInformerFactory.Cluster().V1().ManagedClusters().Informer().GetStore()
			for _, cluster := range c.existingManagedClusters {
				if err := clusterStore.Add(cluster); err != nil {
					t.Fatal(err)
				}
			}

			admissionHandler := &AdmissionHandler{
				KubeClient:    kubeClient,
				HiveClient:    hiveClient,
				ClusterLister: clusterInformerFactory.Cluster().V1().ManagedClusters().Lister(),
			}

			actualResponse := admissionHandler.ValidateResource(c.request)

			if !reflect.DeepEqual(actualResponse.Allowed, c.expectedResponse.Allowed) {
				t.Errorf("case: %v,expected %#v but got: %#v", c.name, c.expectedResponse, actualResponse)
			}
		})
	}
}

const (
	localClusterFmt   = `{"apiVersion":"cluster.open-cluster-management.io/v1","kind":"ManagedCluster","metadata":{"labels":{"local-cluster":"true"},"name":"%s"},"spec":{}}`
	noLocalClusterFmt = `{"apiVersion":"cluster.open-cluster-management.io/v1","kind":"ManagedCluster","metadata":{"name":"%s"},"spec":{}}`
)

func TestLocalCluster(t *testing.T) {

	cases := []struct {
		name                    string
		request                 *v1.AdmissionRequest
		existingManagedClusters []runtime.Object
		expectedResponse        *v1.AdmissionResponse
	}{
		{
			name: "local cluster already exists",
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
						Labels: map[string]string{
							"local-cluster": "true",
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Create,
				Name:      "c1",
				Object: runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(localClusterFmt, "c1")),
				},
			},
		},
		{
			name: "local cluster does not exists",
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Create,
				Name:      "c1",
				Object: runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(localClusterFmt, "c1")),
				},
			},
		},
		{
			name: "local cluster does not exists",
			existingManagedClusters: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c0",
						Labels: map[string]string{
							"local-cluster": "true",
						},
					},
				},
			},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Create,
				Name:      "c1",
				Object: runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(noLocalClusterFmt, "c1")),
				},
			},
		},
		{
			name:                    "update local cluster to non local cluster",
			existingManagedClusters: []runtime.Object{},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Update,
				Name:      "c0",
				OldObject: runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(localClusterFmt, "c0")),
				},
				Object: runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(noLocalClusterFmt, "c0")),
				},
			},
		},
		{
			name:                    "update non local cluster to local cluster",
			existingManagedClusters: []runtime.Object{},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: false,
			},
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Update,
				Name:      "c0",
				OldObject: runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(noLocalClusterFmt, "c0")),
				},
				Object: runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(localClusterFmt, "c0")),
				},
			},
		},
		{
			name:                    "update local cluster",
			existingManagedClusters: []runtime.Object{},
			expectedResponse: &v1.AdmissionResponse{
				Allowed: true,
			},
			request: &v1.AdmissionRequest{
				Resource:  managedClustersGVR,
				Operation: v1.Update,
				Name:      "c0",
				OldObject: runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(localClusterFmt, "c0")),
				},
				Object: runtime.RawExtension{
					Raw: []byte(fmt.Sprintf(localClusterFmt, "c0")),
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clusterClient := clusterfake.NewSimpleClientset(c.existingManagedClusters...)
			clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
			clusterStore := clusterInformerFactory.Cluster().V1().ManagedClusters().Informer().GetStore()
			for _, cluster := range c.existingManagedClusters {
				if err := clusterStore.Add(cluster); err != nil {
					t.Fatal(err)
				}
			}

			kubeClient := kubefake.NewClientset()
			kubeClient.PrependReactor(
				"create",
				"subjectaccessreviews",
				func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &authorizationv1.SubjectAccessReview{
						Status: authorizationv1.SubjectAccessReviewStatus{
							Allowed: true,
						},
					}, nil
				},
			)

			admissionHandler := &AdmissionHandler{
				KubeClient:    kubeClient,
				ClusterLister: clusterInformerFactory.Cluster().V1().ManagedClusters().Lister(),
			}

			actualResponse := admissionHandler.ValidateResource(c.request)
			if !reflect.DeepEqual(actualResponse.Allowed, c.expectedResponse.Allowed) {
				t.Errorf("case: %v,expected %#v but got: %#v", c.name, c.expectedResponse, actualResponse)
			}
		})
	}
}
