package project

import (
	"fmt"
	"slices"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog/v2"
)

type projectView struct {
	name    string
	cluster string
}

func listProjects(namespace, name string, obj runtime.Object, userInfo user.Info) []projectView {
	projects := []projectView{}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		klog.Errorf("failed to converter object %v, %v", obj, err)
		return projects
	}

	clusterRoleBinding, found, err := unstructured.NestedMap(u, "spec", "clusterRoleBinding")
	if err != nil {
		klog.Errorf("invalid roleBindings in %s/%s, %v", namespace, name, err)
		return projects
	}
	if found {
		subject, err := toSubject(clusterRoleBinding)
		if err != nil {
			klog.Errorf("failed to get subject %v", err)
			return projects
		}
		if isBoundUser(subject, userInfo) {
			roleName, err := getRoleRefName(clusterRoleBinding)
			if err != nil {
				klog.Errorf("failed to get roleRef in %s/%s on clusterRoleBing, %v", namespace, name, err)
				return projects
			}

			if isKubeVirtRole(roleName) {
				projects = append(projects, projectView{name: "*", cluster: namespace})
				return projects
			}
		}
	}

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
		binding, ok := rb.(map[string]any)
		if !ok {
			klog.Errorf("invalid roleBinding in %s/%s, %v", namespace, name, err)
			continue
		}

		subject, err := toSubject(binding)
		if err != nil {
			klog.Errorf("failed to get subject %v", err)
			continue
		}

		if !isBoundUser(subject, userInfo) {
			continue
		}

		roleName, err := getRoleRefName(binding)
		if err != nil {
			klog.Errorf("failed to get roleRef in %s/%s on roleBing, %v", namespace, name, err)
			continue
		}

		if !isKubeVirtRole(roleName) {
			continue
		}

		ns, found, err := unstructured.NestedString(binding, "namespace")
		if err != nil {
			klog.Errorf("invalid struct for namespace %v, %v", obj, err)
			continue
		}
		if !found {
			klog.Warningf("namespace is not found in %s/%s", namespace, name)
			continue
		}

		klog.Infof("project %s was found from %s/%s for user(groups=%v,name=%s)",
			ns, namespace, name, userInfo.GetGroups(), userInfo.GetName())
		projects = append(projects, projectView{name: ns, cluster: namespace})
	}

	return projects
}

func isBoundUser(subject *rbacv1.Subject, userInfo user.Info) bool {
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

func getRoleRefName(binding map[string]any) (string, error) {
	roleRef, found, err := unstructured.NestedMap(binding, "roleRef")
	if err != nil {
		return "", fmt.Errorf("invalid struct for roleBinding, %w", err)
	}
	if !found {
		return "", fmt.Errorf("roleRef is not found")
	}

	roleName, found, err := unstructured.NestedString(roleRef, "name")
	if err != nil {
		return "", fmt.Errorf("invalid struct for roleRef, %w", err)
	}
	if !found {
		return "", fmt.Errorf("name is not found in roleRef")
	}
	return roleName, nil
}

func isKubeVirtRole(name string) bool {
	return (name == "kubevirt.io:admin" || name == "kubevirt.io:edit" || name == "kubevirt.io:view")
}
