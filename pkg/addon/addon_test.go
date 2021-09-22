package addon

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"open-cluster-management.io/addon-framework/pkg/agent"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func newCluster(name string) *clusterv1.ManagedCluster {
	return &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func newAddon(name, cluster, installNamespace string) *addonapiv1alpha1.ManagedClusterAddOn {
	return &addonapiv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster,
		},
		Spec: *&addonapiv1alpha1.ManagedClusterAddOnSpec{
			InstallNamespace: installNamespace,
		},
	}
}

func TestManifest(t *testing.T) {
	tests := []struct {
		name              string
		agent             *foundationAgent
		cluster           *clusterv1.ManagedCluster
		addon             *addonapiv1alpha1.ManagedClusterAddOn
		expectedNamespace string
		expectedImage     string
	}{
		{
			name:              "no install namespace",
			agent:             NewAgent(nil, "work-mgr", "test", "open-cluster-management-agent-addon"),
			cluster:           newCluster("cluster1"),
			addon:             newAddon("work-mgr", "cluster1", ""),
			expectedNamespace: "open-cluster-management-agent-addon",
			expectedImage:     "test",
		},
		{
			name:              "no install namespace",
			agent:             NewAgent(nil, "work-mgr", "test", "open-cluster-management-agent-addon"),
			cluster:           newCluster("cluster1"),
			addon:             newAddon("work-mgr", "cluster1", "test"),
			expectedNamespace: "test",
			expectedImage:     "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			objects, err := test.agent.Manifests(test.cluster, test.addon)

			if err != nil {
				t.Errorf("failed to get manifests with error %v", err)
			}

			if len(objects) != 5 {
				t.Errorf("objects number is not correct, got %d", len(objects))
			}

			for _, o := range objects {
				switch object := o.(type) {
				case *appsv1.Deployment:
					if object.Namespace != test.expectedNamespace {
						t.Errorf("expected namespace is %s, but got %s", test.expectedNamespace, object.Namespace)
					}
					if object.Spec.Template.Spec.Containers[0].Image != test.expectedImage {
						t.Errorf("expected image is %s, but got %s", test.expectedImage, object.Spec.Template.Spec.Containers[0].Image)
					}
				}
			}
		})
	}
}

func TestCreateOrUpdateRoleBinding(t *testing.T) {
	tests := []struct {
		name            string
		initObjects     []runtime.Object
		clusterName     string
		validateActions func(t *testing.T, actions []clienttesting.Action)
	}{
		{
			name:        "create a new rolebinding",
			initObjects: []runtime.Object{},
			clusterName: "cluster1",
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 2 {
					t.Errorf("expecte 2 actions, but got %v", actions)
				}

				createAction := actions[1].(clienttesting.CreateActionImpl)
				createObject := createAction.Object.(*rbacv1.RoleBinding)

				groups := agent.DefaultGroups("cluster1", "work-mgr")

				if createObject.Subjects[0].Name != groups[0] {
					t.Errorf("Expected group name is %s, but got %s", groups[0], createObject.Subjects[0].Name)
				}
			},
		},
		{
			name: "no update",
			initObjects: []runtime.Object{
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterRoleName,
						Namespace: "cluster1",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:     rbacv1.GroupKind,
							APIGroup: "rbac.authorization.k8s.io",
							Name:     agent.DefaultGroups("cluster1", "work-mgr")[0],
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     clusterRoleName,
					},
				},
			},
			clusterName: "cluster1",
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 1 {
					t.Errorf("expecte 0 actions, but got %v", actions)
				}
			},
		},
		{
			name: "update rolebinding",
			initObjects: []runtime.Object{
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterRoleName,
						Namespace: "cluster1",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:     rbacv1.GroupKind,
							APIGroup: "rbac.authorization.k8s.io",
							Name:     "test",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     clusterRoleName,
					},
				},
			},
			clusterName: "cluster1",
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 2 {
					t.Errorf("expecte 2 actions, but got %v", actions)
				}

				updateAction := actions[1].(clienttesting.UpdateActionImpl)
				updateObject := updateAction.Object.(*rbacv1.RoleBinding)

				groups := agent.DefaultGroups("cluster1", "work-mgr")

				if updateObject.Subjects[0].Name != groups[0] {
					t.Errorf("Expected group name is %s, but got %s", groups[0], updateObject.Subjects[0].Name)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubeClient := fakekube.NewSimpleClientset(test.initObjects...)
			agent := NewAgent(kubeClient, "work-mgr", "test", "open-cluster-management-agent-addon")
			err := agent.createOrUpdateRoleBinding(test.clusterName)
			if err != nil {
				t.Errorf("createOrUpdateRoleBinding expected no error, but got %v", err)
			}

			test.validateActions(t, kubeClient.Actions())
		})
	}
}
