package cache

import (
	"fmt"
	"slices"
	"sync"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	projectv1 "github.com/openshift/api/project/v1"
)

const KubeVirtProjectClusterLabel = "cluster"

type kubeVirtProject struct {
	name    string
	cluster string
}

type KubevirtProjectCache struct {
	sync.RWMutex
	clusterPermissionLister cache.GenericLister
	clusterPermissions      sets.Set[string]
}

func NewKubevirtProjectCache(lister cache.GenericLister) *KubevirtProjectCache {
	return &KubevirtProjectCache{
		clusterPermissionLister: lister,
	}
}

func (c *KubevirtProjectCache) synchronize() {
	objs, err := c.clusterPermissionLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("failed to list cluster permissions %v", err)
	}

	clusterPermissions := sets.Set[string]{}
	for _, obj := range objs {
		o, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			klog.Errorf("failed to converter object %v, %v", obj, err)
			continue
		}

		u := unstructured.Unstructured{Object: o}
		if !isKubeVirtPermission(u.GetName()) {
			continue
		}

		clusterPermissions.Insert(u.GetNamespace() + "/" + u.GetName())
	}

	// update the current clusterPermissions
	c.Lock()
	defer c.Unlock()
	c.clusterPermissions = clusterPermissions
}

func (c *KubevirtProjectCache) List(userInfo user.Info, selector labels.Selector) (*projectv1.ProjectList, error) {
	klog.V(4).Infof("list projects from %v for user(groups=%v,name=%s)", c.clusterPermissions, userInfo.GetGroups(), userInfo.GetName())

	c.RLock()
	defer c.RUnlock()

	allProjects := sets.New[kubeVirtProject]()
	for key := range c.clusterPermissions {
		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			klog.Errorf("failed to split ClusterPermission namespace %s, %v", key, err)
			continue
		}

		obj, err := c.clusterPermissionLister.ByNamespace(namespace).Get(name)
		if err != nil {
			klog.Errorf("failed to get ClusterPermission %s, %v", key, err)
			continue
		}

		// find the projects from ClusterPermission RoleBindings with current user
		allProjects.Insert(listKubeVirtProjects(namespace, name, obj, userInfo)...)
	}

	projects := []projectv1.Project{}
	for p := range allProjects {
		projects = append(projects, projectv1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: p.name,
				Labels: map[string]string{
					KubeVirtProjectClusterLabel: p.cluster,
				},
			},
		})
	}

	return &projectv1.ProjectList{Items: projects}, nil
}

func (c *KubevirtProjectCache) Run(period time.Duration) {
	go wait.Forever(func() { c.synchronize() }, period)
}

func isKubeVirtPermission(name string) bool {
	return (name == "kubevirt-admin" || name == "kubevirt-view" || name == "kubevirt-edit")
}

func listKubeVirtProjects(namespace, name string, obj runtime.Object, userInfo user.Info) []kubeVirtProject {
	projects := []kubeVirtProject{}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		klog.Errorf("failed to converter object %v, %v", obj, err)
		return projects
	}

	clusterRoleBinding, found, err := unstructured.NestedMap(u, "spec", "clusterRoleBinding")
	if err != nil {
		klog.Errorf("invalid clusterRoleBinding in %s/%s, %v", namespace, name, err)
		return projects
	}
	if found {
		if isBoundUser(clusterRoleBinding, userInfo) {
			return []kubeVirtProject{{name: "any", cluster: namespace}}
		}

		// current user is not in clusterRoleBinding, continue to find the projects from roleBindings
	}

	// no clusterRoleBinding, continue to find the projects from roleBindings
	roleBindings, found, err := unstructured.NestedSlice(u, "spec", "roleBindings")
	if err != nil {
		klog.Errorf("invalid roleBindings in %s/%s, %v", namespace, name, err)
		return projects
	}
	if !found {
		// no bindings, do nothing
		return projects
	}

	for _, rb := range roleBindings {
		roleBinding, ok := rb.(map[string]any)
		if !ok {
			klog.Errorf("invalid roleBinding in %s/%s, %v", namespace, name, err)
			continue
		}

		if !isBoundUser(roleBinding, userInfo) {
			continue
		}

		ns, found, err := unstructured.NestedString(roleBinding, "namespace")
		if err != nil {
			klog.Errorf("invalid struct for namespace %v, %v", obj, err)
			continue
		}
		if !found {
			// TODO NamespaceSelector??
			klog.Warningf("namespace is not found in %s/%s", namespace, name)
			continue
		}

		klog.Infof("project %s was found from %s/%s for user(groups=%v,name=%s)",
			ns, namespace, name, userInfo.GetGroups(), userInfo.GetName())
		projects = append(projects, kubeVirtProject{name: ns, cluster: namespace})
	}

	return projects
}

func isBoundUser(binding map[string]any, userInfo user.Info) bool {
	subject, err := toSubject(binding)
	if err != nil {
		klog.Errorf("failed to get subject %v", err)
		return false
	}

	// TODO ServiceAccount and ManagedServiceAccount??
	switch subject.Kind {
	case rbacv1.GroupKind:
		if slices.Contains(userInfo.GetGroups(), subject.Name) {
			return true
		}
	case rbacv1.UserKind:
		return subject.Name == userInfo.GetName()
	}

	return false
}

func toSubject(binding map[string]any) (*rbacv1.Subject, error) {
	u, found, err := unstructured.NestedMap(binding, "subject")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no subject")
	}

	kind, found, err := unstructured.NestedString(u, "kind")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no kind in subject")
	}

	name, found, err := unstructured.NestedString(u, "name")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no name in subject")
	}

	return &rbacv1.Subject{
		Kind: kind,
		Name: name,
	}, nil
}
