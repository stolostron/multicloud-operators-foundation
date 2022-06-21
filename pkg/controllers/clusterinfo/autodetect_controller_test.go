package clusterinfo

import (
	"context"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	clusterclaims "github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestAutoDetectReconciler(existingObjs []runtime.Object) (*AutoDetectReconciler, client.Client) {
	client := fake.NewFakeClientWithScheme(scheme, existingObjs...)
	return &AutoDetectReconciler{
		client: client,
		scheme: scheme,
	}, client
}

func TestAutoDetectReconcile(t *testing.T) {
	ctx := context.Background()
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
							clusterv1beta1.LabelCloudVendor: clusterv1beta1.AutoDetect,
							clusterv1beta1.LabelKubeVendor:  clusterv1beta1.AutoDetect,
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
			requeue:           false,
		},
		{
			name: "UpdateManagedClusterLabels",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
						Labels: map[string]string{
							clusterv1beta1.LabelCloudVendor: clusterv1beta1.AutoDetect,
							clusterv1beta1.LabelKubeVendor:  clusterv1beta1.AutoDetect,
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
			svrc, _ := newTestAutoDetectReconciler(test.existingObjs)
			res, err := svrc.Reconcile(ctx, test.req)
			validateError(t, err, test.expectedErrorType)
			if test.requeue {
				assert.Equal(t, res.Requeue, true)
			} else {
				assert.Equal(t, res.Requeue, false)
			}
		})
	}
}

func TestOSDVendorOcpVersion(t *testing.T) {
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
							clusterv1beta1.LabelCloudVendor: clusterv1beta1.AutoDetect,
							clusterv1beta1.LabelKubeVendor:  clusterv1beta1.AutoDetect,
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
			expectedLabel:     map[string]string{clusterv1beta1.LabelCloudVendor: "Azure", clusterv1beta1.LabelKubeVendor: "OpenShift"},
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
							clusterv1beta1.LabelCloudVendor: clusterv1beta1.AutoDetect,
							clusterv1beta1.LabelKubeVendor:  clusterv1beta1.AutoDetect,
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
			expectedLabel:     map[string]string{clusterv1beta1.LabelCloudVendor: "Azure", clusterv1beta1.LabelKubeVendor: "OpenShift", clusterv1beta1.LabelManagedBy: "platform"},
			expectedErrorType: nil,
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
		},
		{
			name: "UpdateManagedClusterLabelsOpenShiftVersion",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
					},
					Spec: clusterv1.ManagedClusterSpec{},
					Status: clusterv1.ManagedClusterStatus{
						ClusterClaims: []clusterv1.ManagedClusterClaim{
							{
								Name:  clusterclaims.ClaimOpenshiftVersion,
								Value: "4.6",
							},
						},
					},
				},
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedClusterName,
						Namespace: ManagedClusterName,
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
				},
			},
			expectedLabel:     map[string]string{clusterv1beta1.OCPVersion: "4.6"},
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
			ctx := context.Background()
			svrc, client := newTestAutoDetectReconciler(test.existingObjs)
			_, err := svrc.Reconcile(ctx, test.req)
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
