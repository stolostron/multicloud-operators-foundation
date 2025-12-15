package userpermission

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterpermissionv1alpha1 "open-cluster-management.io/cluster-permission/api/v1alpha1"
	cpfake "open-cluster-management.io/cluster-permission/client/clientset/versioned/fake"
	cpinformers "open-cluster-management.io/cluster-permission/client/informers/externalversions"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

var (
	// Discoverable ClusterRoles
	discoverableClusterRole1 = rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "admin-role",
			ResourceVersion: "1",
			Labels: map[string]string{
				clusterviewv1alpha1.DiscoverableClusterRoleLabel: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"*"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
		},
	}

	discoverableClusterRole2 = rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "view-role",
			ResourceVersion: "2",
			Labels: map[string]string{
				clusterviewv1alpha1.DiscoverableClusterRoleLabel: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
		},
	}

	// ClusterRole that grants BOTH managedclusteractions AND managedclusterviews create permissions
	// This represents the actual permissions needed for managedcluster:admin role
	adminClusterRole = rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "managedcluster-admin",
			ResourceVersion: "3",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"action.open-cluster-management.io"},
				Resources: []string{"managedclusteractions"},
			},
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"view.open-cluster-management.io"},
				Resources: []string{"managedclusterviews"},
			},
		},
	}

	// ClusterRole that grants managedclusterviews create permission
	viewClusterRole = rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "managedcluster-view",
			ResourceVersion: "4",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"view.open-cluster-management.io"},
				Resources: []string{"managedclusterviews"},
			},
		},
	}

	// ManagedClusters
	cluster1 = clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "cluster1",
			ResourceVersion: "1",
		},
	}

	cluster2 = clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "cluster2",
			ResourceVersion: "2",
		},
	}

	// ClusterRoleBindings
	crbUser1AdminRole = rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "user1-admin-binding",
			ResourceVersion: "1",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.GroupName,
				Name:     "user1",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "managedcluster-admin",
		},
	}

	crbGroup1ViewRole = rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "group1-view-binding",
			ResourceVersion: "2",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.GroupKind,
				APIGroup: rbacv1.GroupName,
				Name:     "group1",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "managedcluster-view",
		},
	}

	// RoleBindings
	rbUser2AdminInCluster1 = rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "user2-admin-binding",
			Namespace:       "cluster1",
			ResourceVersion: "3",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.GroupName,
				Name:     "user2",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "managedcluster-admin",
		},
	}

	// ClusterPermissions
	cpCluster1 = clusterpermissionv1alpha1.ClusterPermission{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "cluster1-permission",
			Namespace:       "cluster1",
			ResourceVersion: "1",
		},
		Spec: clusterpermissionv1alpha1.ClusterPermissionSpec{
			ClusterRoleBinding: &clusterpermissionv1alpha1.ClusterRoleBinding{
				RoleRef: &rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     "ClusterRole",
					Name:     "admin-role",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:     rbacv1.UserKind,
						APIGroup: rbacv1.GroupName,
						Name:     "user3",
					},
				},
			},
		},
	}

	cpCluster2 = clusterpermissionv1alpha1.ClusterPermission{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "cluster2-permission",
			Namespace:       "cluster2",
			ResourceVersion: "2",
		},
		Spec: clusterpermissionv1alpha1.ClusterPermissionSpec{
			RoleBindings: &[]clusterpermissionv1alpha1.RoleBinding{
				{
					Namespace: "default",
					RoleRef: clusterpermissionv1alpha1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "view-role",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:     rbacv1.GroupKind,
							APIGroup: rbacv1.GroupName,
							Name:     "group2",
						},
					},
				},
			},
		},
	}
)

func setupUserPermissionCache(stopCh chan struct{}) *Cache {
	// Create fake clients
	fakeKubeClient := fake.NewSimpleClientset(
		&discoverableClusterRole1,
		&discoverableClusterRole2,
		&adminClusterRole,
		&viewClusterRole,
		&crbUser1AdminRole,
		&crbGroup1ViewRole,
		&rbUser2AdminInCluster1,
	)

	fakeClusterClient := clusterfake.NewSimpleClientset(&cluster1, &cluster2)
	fakeClusterPermissionClient := cpfake.NewSimpleClientset(&cpCluster1, &cpCluster2)

	// Create informers
	kubeInformers := informers.NewSharedInformerFactory(fakeKubeClient, 10*time.Minute)
	clusterInformers := clusterv1informers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
	cpInformers := cpinformers.NewSharedInformerFactory(fakeClusterPermissionClient, 10*time.Minute)

	// Add objects to informers
	kubeInformers.Rbac().V1().ClusterRoles().Informer().GetIndexer().Add(&discoverableClusterRole1)
	kubeInformers.Rbac().V1().ClusterRoles().Informer().GetIndexer().Add(&discoverableClusterRole2)
	kubeInformers.Rbac().V1().ClusterRoles().Informer().GetIndexer().Add(&adminClusterRole)
	kubeInformers.Rbac().V1().ClusterRoles().Informer().GetIndexer().Add(&viewClusterRole)

	kubeInformers.Rbac().V1().ClusterRoleBindings().Informer().GetIndexer().Add(&crbUser1AdminRole)
	kubeInformers.Rbac().V1().ClusterRoleBindings().Informer().GetIndexer().Add(&crbGroup1ViewRole)

	kubeInformers.Rbac().V1().RoleBindings().Informer().GetIndexer().Add(&rbUser2AdminInCluster1)

	clusterInformers.Cluster().V1().ManagedClusters().Informer().GetIndexer().Add(&cluster1)
	clusterInformers.Cluster().V1().ManagedClusters().Informer().GetIndexer().Add(&cluster2)

	cpInformers.Api().V1alpha1().ClusterPermissions().Informer().GetIndexer().Add(&cpCluster1)
	cpInformers.Api().V1alpha1().ClusterPermissions().Informer().GetIndexer().Add(&cpCluster2)

	// Start informers
	kubeInformers.Start(stopCh)
	clusterInformers.Start(stopCh)
	cpInformers.Start(stopCh)

	// Create cache
	cache := New(
		kubeInformers.Rbac().V1().ClusterRoles(),
		kubeInformers.Rbac().V1().ClusterRoleBindings(),
		kubeInformers.Rbac().V1().Roles(),
		kubeInformers.Rbac().V1().RoleBindings(),
		clusterInformers.Cluster().V1().ManagedClusters().Lister(),
		cpInformers.Api().V1alpha1().ClusterPermissions().Lister(),
	)

	// Trigger initial synchronization
	cache.synchronize()

	return cache
}

func TestUserPermissionCache_List(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cache := setupUserPermissionCache(stopCh)

	tests := []struct {
		name                string
		user                user.Info
		expectedRoles       sets.Set[string]
		expectedClusterDefs map[string]bool // roleName -> should have ClusterRoleDefinition
	}{
		{
			name: "user1 has managedcluster:admin via ClusterRoleBinding",
			user: &user.DefaultInfo{
				Name:   "user1",
				Groups: []string{},
			},
			expectedRoles: sets.New(clusterviewv1alpha1.ManagedClusterAdminRole),
			expectedClusterDefs: map[string]bool{
				clusterviewv1alpha1.ManagedClusterAdminRole: true,
			},
		},
		{
			name: "group1 has managedcluster:view via ClusterRoleBinding",
			user: &user.DefaultInfo{
				Name:   "userInGroup1",
				Groups: []string{"group1"},
			},
			expectedRoles: sets.New(clusterviewv1alpha1.ManagedClusterViewRole),
			expectedClusterDefs: map[string]bool{
				clusterviewv1alpha1.ManagedClusterViewRole: true,
			},
		},
		{
			name: "user2 has managedcluster:admin via RoleBinding in cluster1",
			user: &user.DefaultInfo{
				Name:   "user2",
				Groups: []string{},
			},
			expectedRoles: sets.New(clusterviewv1alpha1.ManagedClusterAdminRole),
			expectedClusterDefs: map[string]bool{
				clusterviewv1alpha1.ManagedClusterAdminRole: true,
			},
		},
		{
			name: "user3 has admin-role via ClusterPermission",
			user: &user.DefaultInfo{
				Name:   "user3",
				Groups: []string{},
			},
			expectedRoles: sets.New("admin-role"),
			expectedClusterDefs: map[string]bool{
				"admin-role": true,
			},
		},
		{
			name: "group2 has view-role via ClusterPermission",
			user: &user.DefaultInfo{
				Name:   "userInGroup2",
				Groups: []string{"group2"},
			},
			expectedRoles: sets.New("view-role"),
			expectedClusterDefs: map[string]bool{
				"view-role": true,
			},
		},
		{
			name: "user without permissions",
			user: &user.DefaultInfo{
				Name:   "noPermUser",
				Groups: []string{},
			},
			expectedRoles:       sets.New[string](),
			expectedClusterDefs: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cache.List(tt.user, labels.Everything())
			assert.NoError(t, err)
			assert.NotNil(t, result)

			actualRoles := sets.New[string]()
			for _, item := range result.Items {
				actualRoles.Insert(item.Name)
			}

			assert.Equal(t, tt.expectedRoles, actualRoles,
				"Expected roles %v, got %v", tt.expectedRoles.UnsortedList(), actualRoles.UnsortedList())

			// Verify ClusterRoleDefinition is set correctly
			for _, item := range result.Items {
				shouldHaveDef, exists := tt.expectedClusterDefs[item.Name]
				if !exists {
					continue
				}
				if shouldHaveDef {
					assert.NotNil(t, item.Status.ClusterRoleDefinition.Rules,
						"Role %s should have ClusterRoleDefinition", item.Name)
					assert.Greater(t, len(item.Status.ClusterRoleDefinition.Rules), 0,
						"Role %s should have non-empty Rules", item.Name)
				}
			}
		})
	}
}

func TestUserPermissionCache_Get(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cache := setupUserPermissionCache(stopCh)

	tests := []struct {
		name          string
		user          user.Info
		roleName      string
		expectError   bool
		expectBinding bool
	}{
		{
			name: "user1 can get managedcluster:admin",
			user: &user.DefaultInfo{
				Name:   "user1",
				Groups: []string{},
			},
			roleName:      clusterviewv1alpha1.ManagedClusterAdminRole,
			expectError:   false,
			expectBinding: true,
		},
		{
			name: "user3 can get admin-role",
			user: &user.DefaultInfo{
				Name:   "user3",
				Groups: []string{},
			},
			roleName:      "admin-role",
			expectError:   false,
			expectBinding: true,
		},
		{
			name: "user without permission cannot get role",
			user: &user.DefaultInfo{
				Name:   "noPermUser",
				Groups: []string{},
			},
			roleName:      "admin-role",
			expectError:   true,
			expectBinding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cache.Get(tt.user, tt.roleName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.roleName, result.Name)

				if tt.expectBinding {
					assert.Greater(t, len(result.Status.Bindings), 0,
						"Expected at least one binding for role %s", tt.roleName)
					assert.NotNil(t, result.Status.ClusterRoleDefinition.Rules,
						"Expected ClusterRoleDefinition for role %s", tt.roleName)
				}
			}
		})
	}
}

func TestUserPermissionCache_MergeBindings(t *testing.T) {
	tests := []struct {
		name             string
		existingBindings []clusterviewv1alpha1.ClusterBinding
		newBinding       clusterviewv1alpha1.ClusterBinding
		expectedCount    int
		expectedScope    clusterviewv1alpha1.BindingScope
	}{
		{
			name:             "add first binding",
			existingBindings: []clusterviewv1alpha1.ClusterBinding{},
			newBinding: clusterviewv1alpha1.ClusterBinding{
				Cluster:    "cluster1",
				Scope:      clusterviewv1alpha1.BindingScopeCluster,
				Namespaces: []string{"*"},
			},
			expectedCount: 1,
			expectedScope: clusterviewv1alpha1.BindingScopeCluster,
		},
		{
			name: "cluster scope overrides namespace scope",
			existingBindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster1",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"ns1"},
				},
			},
			newBinding: clusterviewv1alpha1.ClusterBinding{
				Cluster:    "cluster1",
				Scope:      clusterviewv1alpha1.BindingScopeCluster,
				Namespaces: []string{"*"},
			},
			expectedCount: 1,
			expectedScope: clusterviewv1alpha1.BindingScopeCluster,
		},
		{
			name: "cluster scope blocks namespace scope",
			existingBindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster1",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
			},
			newBinding: clusterviewv1alpha1.ClusterBinding{
				Cluster:    "cluster1",
				Scope:      clusterviewv1alpha1.BindingScopeNamespace,
				Namespaces: []string{"ns1"},
			},
			expectedCount: 1,
			expectedScope: clusterviewv1alpha1.BindingScopeCluster,
		},
		{
			name: "merge namespace scopes",
			existingBindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster1",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"ns1"},
				},
			},
			newBinding: clusterviewv1alpha1.ClusterBinding{
				Cluster:    "cluster1",
				Scope:      clusterviewv1alpha1.BindingScopeNamespace,
				Namespaces: []string{"ns2"},
			},
			expectedCount: 1,
			expectedScope: clusterviewv1alpha1.BindingScopeNamespace,
		},
		{
			name: "add different cluster",
			existingBindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster1",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
			},
			newBinding: clusterviewv1alpha1.ClusterBinding{
				Cluster:    "cluster2",
				Scope:      clusterviewv1alpha1.BindingScopeCluster,
				Namespaces: []string{"*"},
			},
			expectedCount: 2,
			expectedScope: clusterviewv1alpha1.BindingScopeCluster,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeOrAppendBinding(tt.existingBindings, tt.newBinding)
			assert.Equal(t, tt.expectedCount, len(result), "Unexpected binding count")

			if len(result) > 0 {
				// For single cluster tests, verify the scope
				if tt.expectedCount == 1 {
					assert.Equal(t, tt.expectedScope, result[0].Scope, "Unexpected scope")
				}

				// For namespace merge tests, verify namespaces are merged
				if tt.name == "merge namespace scopes" {
					namespaces := sets.New(result[0].Namespaces...)
					assert.True(t, namespaces.Has("ns1") && namespaces.Has("ns2"),
						"Namespaces should be merged")
				}
			}
		})
	}
}

func TestUserPermissionCache_CalculateResourceVersionHash(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cache := setupUserPermissionCache(stopCh)

	// Calculate initial hash
	hash1, err := cache.calculateResourceVersionHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// Calculate again - should be the same
	hash2, err := cache.calculateResourceVersionHash()
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2, "Hash should be deterministic")
}

func TestUserPermissionCache_DiscoverableRoles(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cache := setupUserPermissionCache(stopCh)

	// After synchronization, the cache's permission store should have populated discoverable roles
	cache.lock.RLock()
	roles := cache.permissionStore.getDiscoverableRoles()
	cache.lock.RUnlock()

	assert.Equal(t, 2, len(roles), "Should have 2 discoverable roles")

	roleNames := sets.New[string]()
	for _, role := range roles {
		roleNames.Insert(role.Name)
	}
	assert.True(t, roleNames.Has("admin-role"), "Should have admin-role")
	assert.True(t, roleNames.Has("view-role"), "Should have view-role")
}

func TestUserPermissionCache_SyntheticRoles(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cache := setupUserPermissionCache(stopCh)

	t.Run("synthetic admin role definition", func(t *testing.T) {
		def := cache.getSyntheticAdminRoleDefinition()
		assert.NotNil(t, def.Rules)
		assert.Equal(t, 1, len(def.Rules))
		assert.Contains(t, def.Rules[0].APIGroups, "*")
		assert.Contains(t, def.Rules[0].Resources, "*")
		assert.Contains(t, def.Rules[0].Verbs, "*")
	})

	t.Run("synthetic view role definition", func(t *testing.T) {
		def := cache.getSyntheticViewRoleDefinition()
		assert.NotNil(t, def.Rules)
		assert.Equal(t, 1, len(def.Rules))
		assert.Contains(t, def.Rules[0].APIGroups, "*")
		assert.Contains(t, def.Rules[0].Resources, "*")
		assert.Contains(t, def.Rules[0].Verbs, "get")
		assert.Contains(t, def.Rules[0].Verbs, "list")
		assert.Contains(t, def.Rules[0].Verbs, "watch")
	})
}

func TestUserPermissionCache_PermissionGrantChecks(t *testing.T) {
	t.Run("clusterRoleGrantsPermission for managedclusteractions", func(t *testing.T) {
		grants := clusterRoleGrantsPermission(&adminClusterRole, "managedclusteractions",
			"action.open-cluster-management.io")
		assert.True(t, grants, "adminClusterRole should grant managedclusteractions permission")
	})

	t.Run("clusterRoleGrantsPermission for managedclusterviews", func(t *testing.T) {
		grants := clusterRoleGrantsPermission(&viewClusterRole, "managedclusterviews",
			"view.open-cluster-management.io")
		assert.True(t, grants, "viewClusterRole should grant managedclusterviews permission")
	})

	t.Run("clusterRoleGrantsPermission returns false for wrong resource", func(t *testing.T) {
		grants := clusterRoleGrantsPermission(&adminClusterRole, "wrongresource",
			"action.open-cluster-management.io")
		assert.False(t, grants, "adminClusterRole should not grant permission for wrong resource")
	})

	t.Run("clusterRoleGrantsPermission returns false for wrong api group", func(t *testing.T) {
		grants := clusterRoleGrantsPermission(&adminClusterRole, "managedclusteractions",
			"wrong.api.group")
		assert.False(t, grants, "adminClusterRole should not grant permission for wrong API group")
	})
}

func TestUserPermissionCache_Synchronize(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cache := setupUserPermissionCache(stopCh)

	// Verify initial synchronization happened
	assert.NotEmpty(t, cache.resourceVersionHash, "Resource version hash should be set after synchronization")

	cache.lock.RLock()
	roles := cache.permissionStore.getDiscoverableRoles()
	cache.lock.RUnlock()

	assert.Equal(t, 2, len(roles), "Should have 2 discoverable roles")

	// Test idempotency - synchronize again should not change hash
	initialHash := cache.resourceVersionHash
	cache.synchronize()
	assert.Equal(t, initialHash, cache.resourceVersionHash,
		"Hash should not change when resources haven't changed")
}

func TestUserPermissionCache_UserAndGroupPermissions(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	cache := setupUserPermissionCache(stopCh)

	t.Run("user with multiple group memberships gets combined permissions", func(t *testing.T) {
		user := &user.DefaultInfo{
			Name:   "multiGroupUser",
			Groups: []string{"group1", "group2"},
		}

		result, err := cache.List(user, labels.Everything())
		assert.NoError(t, err)

		roleNames := sets.New[string]()
		for _, item := range result.Items {
			roleNames.Insert(item.Name)
		}

		// Should have permissions from both group1 and group2
		assert.True(t, roleNames.Has(clusterviewv1alpha1.ManagedClusterViewRole),
			"Should have view role from group1")
		assert.True(t, roleNames.Has("view-role"), "Should have view-role from group2")
	})

	t.Run("user with both individual and group permissions gets deduplicated permissions", func(t *testing.T) {
		// user1 has managedcluster:admin via direct ClusterRoleBinding
		// user1 also belongs to group1 which has managedcluster:view via ClusterRoleBinding
		// The deduplication logic should remove managedcluster:view since admin is a superset
		user := &user.DefaultInfo{
			Name:   "user1",
			Groups: []string{"group1"},
		}

		result, err := cache.List(user, labels.Everything())
		assert.NoError(t, err)

		roleNames := sets.New[string]()
		for _, item := range result.Items {
			roleNames.Insert(item.Name)
		}

		// Should only have admin permission (view is deduplicated since admin is a superset)
		assert.True(t, roleNames.Has(clusterviewv1alpha1.ManagedClusterAdminRole),
			"Should have admin role from user's direct ClusterRoleBinding")
		assert.False(t, roleNames.Has(clusterviewv1alpha1.ManagedClusterViewRole),
			"Should NOT have view role - it should be deduplicated since admin is a superset")
		assert.Equal(t, 1, roleNames.Len(), "Should have exactly 1 role after deduplication")

		// Verify the admin role has proper ClusterRoleDefinition
		for _, item := range result.Items {
			assert.NotNil(t, item.Status.ClusterRoleDefinition.Rules,
				"Role %s should have ClusterRoleDefinition", item.Name)
			assert.Greater(t, len(item.Status.ClusterRoleDefinition.Rules), 0,
				"Role %s should have non-empty Rules", item.Name)
		}
	})

	t.Run("user with only view permission is not affected by deduplication", func(t *testing.T) {
		// A user belonging to group1 which has managedcluster:view via ClusterRoleBinding
		// Since there's no admin permission, view should still be shown
		user := &user.DefaultInfo{
			Name:   "someViewUser",
			Groups: []string{"group1"},
		}

		result, err := cache.List(user, labels.Everything())
		assert.NoError(t, err)

		roleNames := sets.New[string]()
		for _, item := range result.Items {
			roleNames.Insert(item.Name)
		}

		// Should have view permission (no admin to deduplicate it)
		assert.True(t, roleNames.Has(clusterviewv1alpha1.ManagedClusterViewRole),
			"Should have view role from group1 membership")
		assert.False(t, roleNames.Has(clusterviewv1alpha1.ManagedClusterAdminRole),
			"Should NOT have admin role")
		assert.Equal(t, 1, roleNames.Len(), "Should have exactly 1 role (view)")
	})
}

func TestUserPermissionCache_NilRoleRef(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	// Create a ClusterPermission with nil RoleRef (using inline ClusterRole)
	cpWithNilRoleRef := clusterpermissionv1alpha1.ClusterPermission{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "cluster1-nil-roleref",
			Namespace:       "cluster1",
			ResourceVersion: "3",
		},
		Spec: clusterpermissionv1alpha1.ClusterPermissionSpec{
			ClusterRole: &clusterpermissionv1alpha1.ClusterRole{
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"create"},
						APIGroups: []string{"apps"},
						Resources: []string{"services"},
					},
				},
			},
			ClusterRoleBindings: &[]clusterpermissionv1alpha1.ClusterRoleBinding{
				{
					// RoleRef is nil when using inline ClusterRole
					RoleRef: nil,
					Subject: &rbacv1.Subject{
						Kind:      rbacv1.ServiceAccountKind,
						Name:      "test-sa",
						Namespace: "test-namespace",
					},
				},
			},
		},
	}

	// Create fake clients
	fakeKubeClient := fake.NewSimpleClientset(
		&discoverableClusterRole1,
		&discoverableClusterRole2,
	)
	fakeClusterClient := clusterfake.NewSimpleClientset(&cluster1)
	fakeClusterPermissionClient := cpfake.NewSimpleClientset(&cpWithNilRoleRef)

	// Create informers
	kubeInformers := informers.NewSharedInformerFactory(fakeKubeClient, 10*time.Minute)
	clusterInformers := clusterv1informers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
	cpInformers := cpinformers.NewSharedInformerFactory(fakeClusterPermissionClient, 10*time.Minute)

	// Add objects to informers
	kubeInformers.Rbac().V1().ClusterRoles().Informer().GetIndexer().Add(&discoverableClusterRole1)
	kubeInformers.Rbac().V1().ClusterRoles().Informer().GetIndexer().Add(&discoverableClusterRole2)
	clusterInformers.Cluster().V1().ManagedClusters().Informer().GetIndexer().Add(&cluster1)
	cpInformers.Api().V1alpha1().ClusterPermissions().Informer().GetIndexer().Add(&cpWithNilRoleRef)

	// Start informers
	kubeInformers.Start(stopCh)
	clusterInformers.Start(stopCh)
	cpInformers.Start(stopCh)

	// Create cache
	cache := New(
		kubeInformers.Rbac().V1().ClusterRoles(),
		kubeInformers.Rbac().V1().ClusterRoleBindings(),
		kubeInformers.Rbac().V1().Roles(),
		kubeInformers.Rbac().V1().RoleBindings(),
		clusterInformers.Cluster().V1().ManagedClusters().Lister(),
		cpInformers.Api().V1alpha1().ClusterPermissions().Lister(),
	)

	// This should not panic - the fix handles nil RoleRef gracefully
	t.Run("synchronize does not panic with nil RoleRef", func(t *testing.T) {
		assert.NotPanics(t, func() {
			cache.synchronize()
		}, "synchronize should not panic when ClusterPermission has nil RoleRef")
	})

	// Verify the cache is still functional
	t.Run("cache is functional after processing nil RoleRef", func(t *testing.T) {
		user := &user.DefaultInfo{
			Name:   "test-user",
			Groups: []string{},
		}

		result, err := cache.List(user, labels.Everything())
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// User should have no permissions since the ClusterPermission with nil RoleRef was skipped
		assert.Equal(t, 0, len(result.Items))
	})
}
