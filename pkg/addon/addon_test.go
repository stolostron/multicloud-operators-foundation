package addon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	apiconstants "github.com/stolostron/cluster-lifecycle-api/constants"
	"github.com/stolostron/cluster-lifecycle-api/imageregistry/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	kubeinformers "k8s.io/client-go/informers"
	fakekube "k8s.io/client-go/kubernetes/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clienttesting "k8s.io/client-go/testing"
	"open-cluster-management.io/addon-framework/pkg/addonfactory"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/utils"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = routev1.Install(scheme)
}

const (
	addonImage = "quay.io/stolostron/multicloud-manager:2.5.0"
)

func newAnnotationRegistries(registries []v1alpha1.Registries, namespacePullSecret string) string {
	registriesData := v1alpha1.ImageRegistries{
		PullSecret: namespacePullSecret,
		Registries: registries,
	}

	registriesDataStr, _ := json.Marshal(registriesData)
	return string(registriesDataStr)
}

func newCluster(name string, product string, labels map[string]string, annotations map[string]string) *clusterv1.ManagedCluster {
	cluster := &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
	}
	if product != "" {
		cluster.Status = clusterv1.ManagedClusterStatus{
			ClusterClaims: []clusterv1.ManagedClusterClaim{
				{
					Name:  "product.open-cluster-management.io",
					Value: product,
				},
			},
		}
	}
	return cluster
}

func newAddon(name, cluster, installNamespace string, annotationValues string) *addonapiv1alpha1.ManagedClusterAddOn {
	addon := &addonapiv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster,
		},
		Spec: *&addonapiv1alpha1.ManagedClusterAddOnSpec{
			InstallNamespace: installNamespace,
		},
	}
	if annotationValues != "" {
		addon.SetAnnotations(map[string]string{"addon.open-cluster-management.io/values": annotationValues})
	}
	return addon
}

func newAddonWithCustomizedAnnotation(name, cluster, installNamespace string, annotations map[string]string) *addonapiv1alpha1.ManagedClusterAddOn {
	addon := &addonapiv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cluster,
			Annotations: annotations,
		},
		Spec: *&addonapiv1alpha1.ManagedClusterAddOnSpec{
			InstallNamespace: installNamespace,
		},
	}
	return addon
}

func newAgentAddon(t *testing.T, cluster *clusterv1.ManagedCluster) agent.AgentAddon {
	registrationOption := NewRegistrationOption(nil, nil, WorkManagerAddonName)

	scheme := runtime.NewScheme()
	clusterv1.Install(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).Build()
	getter := newTestConfigGetter(nil)
	getValuesFunc := NewGetValuesFunc(addonImage)
	agentAddon, err := addonfactory.NewAgentAddonFactory(WorkManagerAddonName, ChartFS, ChartDir).
		WithScheme(scheme).
		WithGetValuesFuncs(getValuesFunc, addonfactory.GetValuesFromAddonAnnotation).
		WithAgentRegistrationOption(registrationOption).
		WithAgentHostedInfoFn(HostedClusterInfo).
		WithAgentInstallNamespace(AddonInstallNamespaceFunc(getter, client)).
		BuildHelmAgentAddon()
	if err != nil {
		t.Fatalf("failed to build agent %v", err)

	}
	return agentAddon
}

func output(t *testing.T, name string, objects ...runtime.Object) {
	tmpDir, err := os.MkdirTemp("./", name)
	if err != nil {
		t.Fatalf("failed to create temp %v", err)
	}

	for i, o := range objects {
		data, err := yaml.Marshal(o)
		if err != nil {
			t.Fatalf("failed yaml marshal %v", err)
		}

		err = os.WriteFile(fmt.Sprintf("%v/%v-%v.yaml", tmpDir, i, o.GetObjectKind().GroupVersionKind().Kind), data, 0644)
		if err != nil {
			t.Fatalf("failed to Marshal object.%v", err)
		}

	}
}

func TestManifest(t *testing.T) {
	tests := []struct {
		name                            string
		cluster                         *clusterv1.ManagedCluster
		addon                           *addonapiv1alpha1.ManagedClusterAddOn
		imageRegistry                   *v1alpha1.ManagedClusterImageRegistry
		expectedNamespace               string
		expectedNamespaceOrphaned       bool
		expectedImage                   string
		expectServiceType               v1.ServiceType
		expectedNodeSelector            bool
		expectedClusterRoleBindingNames sets.Set[string]
		expectedCount                   int
	}{
		{
			name:              "is OCP",
			cluster:           newCluster("cluster1", "OpenShift", map[string]string{}, map[string]string{}),
			addon:             newAddon("work-manager", "cluster1", "", `{"global":{"imageOverrides":{"multicloud_manager":"quay.io/test/multicloud_manager:test"}}}`),
			expectedNamespace: "open-cluster-management-agent-addon",
			expectedImage:     "quay.io/test/multicloud_manager:test",
			expectedClusterRoleBindingNames: sets.New[string](
				"open-cluster-management:klusterlet-addon-workmgr",
				"open-cluster-management:klusterlet-addon-workmgr-log",
			),
			expectedCount: 8,
		},
		{
			name:              "is OCP but hub cluster",
			cluster:           newCluster("local-cluster", "OpenShift", map[string]string{}, map[string]string{}),
			addon:             newAddon("work-manager", "local-cluster", "", `{"global":{"nodeSelector":{"node-role.kubernetes.io/infra":""},"imageOverrides":{"multicloud_manager":"quay.io/test/multicloud_manager:test"}}}`),
			expectedNamespace: "open-cluster-management-agent-addon",
			expectedImage:     "quay.io/test/multicloud_manager:test",
			expectedClusterRoleBindingNames: sets.New[string](
				"open-cluster-management:klusterlet-addon-workmgr",
				"open-cluster-management:klusterlet-addon-workmgr-log",
			),
			expectedNodeSelector: true,
			expectedCount:        8,
		},

		{
			name:              "is not OCP",
			cluster:           newCluster("cluster1", "IKS", map[string]string{}, map[string]string{}),
			addon:             newAddon("work-manager", "cluster1", "test", ""),
			expectedNamespace: "test",
			expectedImage:     "quay.io/stolostron/multicloud-manager:2.5.0",
			expectedClusterRoleBindingNames: sets.New[string](
				"open-cluster-management:klusterlet-addon-workmgr:test",
				"open-cluster-management:klusterlet-addon-workmgr-log:test",
			),
			expectServiceType: v1.ServiceTypeLoadBalancer,
			expectedCount:     9,
		},
		{
			name: "imageOverride",
			cluster: newCluster("cluster1", "OpenShift",
				map[string]string{v1alpha1.ClusterImageRegistryLabel: "ns1.imageRegistry1"},
				map[string]string{annotationNodeSelector: "{\"node-role.kubernetes.io/infra\":\"\"}",
					v1alpha1.ClusterImageRegistriesAnnotation: newAnnotationRegistries([]v1alpha1.Registries{
						{Source: "quay.io/stolostron", Mirror: "quay.io/test"},
					}, "")}),
			addon:             newAddon("work-manager", "cluster1", "", ""),
			expectedNamespace: "open-cluster-management-agent-addon",
			expectedImage:     "quay.io/test/multicloud-manager:2.5.0",
			expectedClusterRoleBindingNames: sets.New[string](
				"open-cluster-management:klusterlet-addon-workmgr",
				"open-cluster-management:klusterlet-addon-workmgr-log",
			),
			expectedCount: 8,
		},
		{
			name: "local cluster imageOverride",
			cluster: newCluster("local-cluster-test", "OpenShift",
				map[string]string{
					v1alpha1.ClusterImageRegistryLabel:      "ns1.imageRegistry1",
					apiconstants.SelfManagedClusterLabelKey: "true",
				},
				map[string]string{annotationNodeSelector: "{\"node-role.kubernetes.io/infra\":\"\"}",
					v1alpha1.ClusterImageRegistriesAnnotation: newAnnotationRegistries([]v1alpha1.Registries{
						{Source: "quay.io/stolostron", Mirror: "quay.io/test"},
					}, "")}),
			addon:             newAddon("work-manager", "local-cluster-test", "", ""),
			expectedNamespace: "open-cluster-management-agent-addon",
			expectedImage:     "quay.io/test/multicloud-manager:2.5.0",
			expectedClusterRoleBindingNames: sets.New[string](
				"open-cluster-management:klusterlet-addon-workmgr",
				"open-cluster-management:klusterlet-addon-workmgr-log",
			),
			expectedNodeSelector: true,
			expectedCount:        8,
		},
		{
			name: "hosted mode",
			cluster: newCluster("local-cluster-test", "OpenShift",
				map[string]string{
					v1alpha1.ClusterImageRegistryLabel:      "ns1.imageRegistry1",
					apiconstants.SelfManagedClusterLabelKey: "true",
				},
				map[string]string{annotationNodeSelector: "{\"node-role.kubernetes.io/infra\":\"\"}",
					v1alpha1.ClusterImageRegistriesAnnotation: newAnnotationRegistries([]v1alpha1.Registries{
						{Source: "quay.io/stolostron", Mirror: "quay.io/test"},
					}, ""),
					apiconstants.AnnotationKlusterletHostingClusterName: "cluster2",
					apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
					AnnotationEnableHostedModeAddons:                    "true",
				}),
			addon: newAddonWithCustomizedAnnotation(
				"work-manager", "local-cluster-test", "", map[string]string{}),
			expectedNamespace:    "klusterlet-local-cluster-test",
			expectedImage:        "quay.io/test/multicloud-manager:2.5.0",
			expectedNodeSelector: true,
			expectedClusterRoleBindingNames: sets.New[string](
				"open-cluster-management:klusterlet-addon-workmgr:klusterlet-local-cluster-test",
			),
			expectedCount: 7,
		},
		{
			name: "hosted mode in klusterlet agent namespace",
			cluster: newCluster("cluster1", "OpenShift",
				map[string]string{v1alpha1.ClusterImageRegistryLabel: "ns1.imageRegistry1"},
				map[string]string{annotationNodeSelector: "{\"node-role.kubernetes.io/infra\":\"\"}",
					v1alpha1.ClusterImageRegistriesAnnotation: newAnnotationRegistries([]v1alpha1.Registries{
						{Source: "quay.io/stolostron", Mirror: "quay.io/test"},
					}, ""),
					apiconstants.AnnotationKlusterletHostingClusterName: "cluster2",
					apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
					AnnotationEnableHostedModeAddons:                    "true",
				}),
			addon: newAddonWithCustomizedAnnotation(
				"work-manager", "cluster1", "klusterlet-cluster1", map[string]string{}),
			expectedNamespace:         "klusterlet-cluster1",
			expectedNamespaceOrphaned: true,
			expectedImage:             "quay.io/test/multicloud-manager:2.5.0",
			expectedClusterRoleBindingNames: sets.New[string](
				"open-cluster-management:klusterlet-addon-workmgr:klusterlet-cluster1",
			),
			expectedCount: 7,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			agentAddon := newAgentAddon(t, test.cluster)
			objects, err := agentAddon.Manifests(test.cluster, test.addon)
			if err != nil {
				t.Errorf("failed to get manifests with error %v", err)
			}

			if len(objects) != test.expectedCount {
				t.Errorf("expected objects number is %d, got %d, %v", test.expectedCount, len(objects), objects)
			}

			clusterRoleNames := sets.New[string]()
			for _, o := range objects {
				switch object := o.(type) {
				case *appsv1.Deployment:
					if object.Namespace != test.expectedNamespace {
						t.Errorf("expected namespace is %s, but got %s", test.expectedNamespace, object.Namespace)
					}
					if object.Spec.Template.Spec.Containers[0].Image != test.expectedImage {
						t.Errorf("expected image is %s, but got %s", test.expectedImage, object.Spec.Template.Spec.Containers[0].Image)
					}
					if test.expectedNodeSelector && len(object.Spec.Template.Spec.NodeSelector) == 0 {
						t.Errorf("expected nodeSelector, but got empty")
					}
					if !test.expectedNodeSelector && len(object.Spec.Template.Spec.NodeSelector) != 0 {
						t.Errorf("expected no nodeSelector, but got it.")
					}

					hostedMode := false
					if v, ok := test.cluster.GetAnnotations()[addonapiv1alpha1.HostingClusterNameAnnotationKey]; ok && v != "" {
						hostedMode = true
					}
					if hostedMode {
						enabelSyncLabelsToClusterclaims := true
						for _, arg := range object.Spec.Template.Spec.Containers[0].Args {
							if arg == "--enable-sync-labels-to-clusterclaims=false" {
								enabelSyncLabelsToClusterclaims = false
							}
						}
						if enabelSyncLabelsToClusterclaims {
							t.Errorf("%s expected --enable-sync-labels-to-clusterclaims=false, but got true.", test.name)
						}
					}
				case *v1.Service:
					if object.Spec.Type != test.expectServiceType {
						t.Errorf("expected service type is %s, but got %s ", test.expectServiceType, object.Spec.Type)
					}
				case *rbacv1.ClusterRoleBinding:
					clusterRoleNames.Insert(object.Name)
				}
			}

			if !test.expectedClusterRoleBindingNames.Equal(clusterRoleNames) {
				t.Errorf("expected clusterrolebinding name is not right got, %v", clusterRoleNames)
			}
			// output is for debug
			// output(t, test.name, objects...)
		})
	}
}

type testConfigGetter struct {
	config *addonapiv1alpha1.AddOnDeploymentConfig
}

func newTestConfigGetter(config *addonapiv1alpha1.AddOnDeploymentConfig) *testConfigGetter {
	return &testConfigGetter{
		config: config,
	}
}

func (t *testConfigGetter) Get(_ context.Context, _ string, _ string) (*addonapiv1alpha1.AddOnDeploymentConfig, error) {
	if t.config == nil {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: "addon.open-cluster-management.io", Resource: "addondeploymentconfigs"},
			"",
		)
	}
	return t.config, nil
}

func newAddonConfigWithNamespace(namespace string) *addonapiv1alpha1.AddOnDeploymentConfig {
	return &addonapiv1alpha1.AddOnDeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "test",
		},
		Spec: addonapiv1alpha1.AddOnDeploymentConfigSpec{
			AgentInstallNamespace: namespace,
		},
	}
}

func newAddonWithConfig(name, cluster, annotationValue string, config *addonapiv1alpha1.AddOnDeploymentConfig) *addonapiv1alpha1.ManagedClusterAddOn {
	specHash, _ := utils.GetAddOnDeploymentConfigSpecHash(config)
	addon := newAddon(name, cluster, "", annotationValue)
	addon.Status = addonapiv1alpha1.ManagedClusterAddOnStatus{
		ConfigReferences: []addonapiv1alpha1.ConfigReference{
			{
				ConfigGroupResource: addonapiv1alpha1.ConfigGroupResource{
					Group:    utils.AddOnDeploymentConfigGVR.Group,
					Resource: utils.AddOnDeploymentConfigGVR.Resource,
				},
				DesiredConfig: &addonapiv1alpha1.ConfigSpecHash{
					SpecHash: specHash,
				},
			},
		},
	}
	return addon
}

func TestAddonInstallNamespaceFunc(t *testing.T) {
	tests := []struct {
		name              string
		addon             *addonapiv1alpha1.ManagedClusterAddOn
		cluster           *clusterv1.ManagedCluster
		config            *addonapiv1alpha1.AddOnDeploymentConfig
		expectedNamespace string
	}{
		{
			name:              "default namespace",
			addon:             newAddon("test", "cluster1", "", ""),
			cluster:           newCluster("cluster1", "", map[string]string{}, map[string]string{}),
			expectedNamespace: "",
		},
		{
			name:              "customized namespace",
			addon:             newAddonWithConfig("test", "cluster1", "", newAddonConfigWithNamespace("ns1")),
			cluster:           newCluster("cluster1", "", map[string]string{}, map[string]string{}),
			config:            newAddonConfigWithNamespace("ns1"),
			expectedNamespace: "ns1",
		},
		{
			name:  "hosted mode disabled",
			addon: newAddonWithConfig("test", "cluster1", "", newAddonConfigWithNamespace("ns1")),
			cluster: newCluster("cluster1", "", map[string]string{},
				map[string]string{
					apiconstants.AnnotationKlusterletHostingClusterName: "cluster2",
					apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
				}),
			config:            newAddonConfigWithNamespace("ns1"),
			expectedNamespace: "ns1",
		},
		{
			name:  "hosted mode",
			addon: newAddonWithConfig("test", "cluster1", "", newAddonConfigWithNamespace("ns1")),
			cluster: newCluster("cluster1", "", map[string]string{},
				map[string]string{
					apiconstants.AnnotationKlusterletHostingClusterName: "cluster2",
					apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
					AnnotationEnableHostedModeAddons:                    "true",
				}),
			config:            newAddonConfigWithNamespace("ns1"),
			expectedNamespace: "klusterlet-cluster1",
		},
	}

	scheme := runtime.NewScheme()
	clusterv1.Install(scheme)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(test.cluster).Build()
			getter := newTestConfigGetter(test.config)
			nsFunc := AddonInstallNamespaceFunc(getter, client)
			namespace, _ := nsFunc(test.addon)
			if namespace != test.expectedNamespace {
				t.Errorf("namespace should be %s, but get %s", test.expectedNamespace, namespace)
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
				if len(actions) != 1 {
					t.Errorf("expecte 2 actions, but got %v", actions)
				}

				createAction := actions[0].(clienttesting.CreateActionImpl)
				createObject := createAction.Object.(*rbacv1.RoleBinding)

				groups := agent.DefaultGroups("cluster1", "work-manager")

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
							Name:     agent.DefaultGroups("cluster1", "work-manager")[0],
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
				if len(actions) != 0 {
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
						Name:     "test",
					},
				},
			},
			clusterName: "cluster1",
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 1 {
					t.Errorf("expecte 2 actions, but got %v", actions)
				}

				updatedAction := actions[0].(clienttesting.UpdateActionImpl)
				updateObject := updatedAction.Object.(*rbacv1.RoleBinding)

				groups := agent.DefaultGroups("cluster1", "work-manager")

				if updateObject.Subjects[0].Name != groups[0] {
					t.Errorf("Expected group name is %s, but got %s", groups[0], updateObject.Subjects[0].Name)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubeClient := fakekube.NewSimpleClientset(test.initObjects...)
			kubeInformers := kubeinformers.NewSharedInformerFactory(kubeClient, 10*time.Minute)
			for _, obj := range test.initObjects {
				kubeInformers.Rbac().V1().RoleBindings().Informer().GetStore().Add(obj)
			}
			opt := NewRegistrationOption(kubeClient, kubeInformers.Rbac().V1().RoleBindings(), "work-manager")
			err := opt.PermissionConfig(
				&clusterv1.ManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: test.clusterName}},
				&addonapiv1alpha1.ManagedClusterAddOn{ObjectMeta: metav1.ObjectMeta{Name: "work-manager"}},
			)
			if err != nil {
				t.Errorf("createOrUpdateRoleBinding expected no error, but got %v", err)
			}

			test.validateActions(t, kubeClient.Actions())
		})
	}
}
