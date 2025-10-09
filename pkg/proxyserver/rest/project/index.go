package project

import (
	"fmt"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
)

const ClusterPermissionSubjectIndexKey = "clusterpermissionsubjects"

func IndexClusterPermissionBySubject(obj any) ([]string, error) {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		klog.Errorf("failed to converter object %v, %v", obj, err)
		return nil, fmt.Errorf("")
	}

	o := unstructured.Unstructured{Object: u}
	namespace := o.GetNamespace()
	name := o.GetName()

	keySet := sets.New[string]()

	clusterRoleBinding, found, err := unstructured.NestedMap(u, "spec", "clusterRoleBinding")
	if err != nil {
		return nil, fmt.Errorf("invalid clusterRoleBinding in %s/%s, %v", namespace, name, err)
	}
	if found {
		subjects, err := toSubjects(clusterRoleBinding)
		if err != nil {
			return nil, fmt.Errorf("failed to find subject from clusterRoleBinding in %s/%s, %v", namespace, name, err)
		}

		for _, subject := range subjects {
			keySet.Insert(fmt.Sprintf("%s/%s/%s/%s", namespace, name, subject.Kind, subject.Name))
		}
	}

	clusterRoleBindings, found, err := unstructured.NestedSlice(u, "spec", "clusterRoleBindings")
	if err != nil {
		return nil, fmt.Errorf("invalid clusterRoleBindings in %s/%s, %v", namespace, name, err)
	}
	if found {
		for _, crb := range clusterRoleBindings {
			binding, ok := crb.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid clusterRoleBinding in %s/%s", namespace, name)
			}

			subjects, err := toSubjects(binding)
			if err != nil {
				return nil, fmt.Errorf("failed to find subject from clusterRoleBindings in %s/%s, %v", namespace, name, err)
			}

			for _, subject := range subjects {
				keySet.Insert(fmt.Sprintf("%s/%s/%s/%s", namespace, name, subject.Kind, subject.Name))
			}
		}
	}

	roleBindings, found, err := unstructured.NestedSlice(u, "spec", "roleBindings")
	if err != nil {
		return nil, fmt.Errorf("invalid roleBindings in %s/%s, %v", namespace, name, err)
	}
	if !found {
		// no bindings, do nothing
		return toKeys(keySet), nil
	}

	for _, rb := range roleBindings {
		binding, ok := rb.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid roleBinding in %s/%s, %v", namespace, name, err)
		}

		subjects, err := toSubjects(binding)
		if err != nil {
			return nil, fmt.Errorf("failed to find subject from roleBindings in %s/%s, %v", namespace, name, err)
		}

		for _, subject := range subjects {
			keySet.Insert(fmt.Sprintf("%s/%s/%s/%s", namespace, name, subject.Kind, subject.Name))
		}
	}

	return toKeys(keySet), nil
}

func toSubjects(binding map[string]any) ([]*rbacv1.Subject, error) {
	subjects := []*rbacv1.Subject{}
	u, hasSubject, err := unstructured.NestedMap(binding, "subject")
	if err != nil {
		return nil, err
	}

	ss, hasSubjects, err := unstructured.NestedSlice(binding, "subjects")
	if err != nil {
		return nil, err
	}

	if !hasSubject && !hasSubjects {
		return nil, fmt.Errorf("no subject or subjects field")
	}

	// If both subject and subjects exist then only subjects will be used.
	if hasSubjects {
		for _, s := range ss {
			su, ok := s.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid subject in subjects")
			}
			sObj, err := toSubject(su)
			if err != nil {
				return nil, err
			}
			subjects = append(subjects, sObj)
			if len(subjects) == 0 {
				return nil, fmt.Errorf("no subject in subjects field")
			}
		}
		return subjects, nil
	}

	sObj, err := toSubject(u)
	if err != nil {
		return nil, err
	}
	return append(subjects, sObj), nil
}

func toSubject(u map[string]any) (*rbacv1.Subject, error) {
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
	return &rbacv1.Subject{Kind: kind, Name: name}, nil
}

func splitKey(key string) (string, string, *rbacv1.Subject, error) {
	slices := strings.SplitN(key, "/", 4)
	if len(slices) != 4 {
		return "", "", nil, fmt.Errorf("failed to split key %s", key)
	}

	namespace := slices[0]
	name := slices[1]
	subject := &rbacv1.Subject{
		Kind: slices[2],
		Name: slices[3],
	}

	return namespace, name, subject, nil
}

func toKeys(keySet sets.Set[string]) []string {
	keys := []string{}
	for key := range keySet {
		keys = append(keys, key)
	}
	return keys
}
