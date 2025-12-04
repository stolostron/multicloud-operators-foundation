package userpermission

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/cache/userpermission"
)

type REST struct {
	cache userpermission.Lister
}

// NewREST returns a RESTStorage object that will work against UserPermission resources
func NewREST(cache userpermission.Lister) *REST {
	return &REST{
		cache: cache,
	}
}

// New returns a new UserPermission
func (s *REST) New() runtime.Object {
	return &clusterviewv1alpha1.UserPermission{}
}

func (s *REST) Destroy() {
}

func (s *REST) NamespaceScoped() bool {
	return false
}

// NewList returns a new UserPermission list
func (*REST) NewList() runtime.Object {
	return &clusterviewv1alpha1.UserPermissionList{}
}

var _ = rest.Lister(&REST{})

// List retrieves a list of UserPermissions for the current user
func (s *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(clusterviewv1alpha1.Resource("userpermissions"), "", fmt.Errorf("unable to list userpermissions without a user on the context"))
	}

	selector := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		selector = options.LabelSelector
	}

	return s.cache.List(user, selector)
}

func (s *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name", Description: "Name of the ClusterRole"},
			{Name: "Bindings", Type: "string", Description: "Cluster bindings in format: cluster(namespaces) or cluster(*) for cluster-scoped"},
		},
	}

	switch obj := object.(type) {
	case *clusterviewv1alpha1.UserPermission:
		table.Rows = append(table.Rows, s.convertUserPermissionToTableRow(obj))
	case *clusterviewv1alpha1.UserPermissionList:
		for i := range obj.Items {
			table.Rows = append(table.Rows, s.convertUserPermissionToTableRow(&obj.Items[i]))
		}
	default:
		return nil, fmt.Errorf("unsupported object type: %T", object)
	}

	return table, nil
}

// convertUserPermissionToTableRow converts a UserPermission to a table row
func (s *REST) convertUserPermissionToTableRow(up *clusterviewv1alpha1.UserPermission) metav1.TableRow {
	// Collect all bindings in cluster(namespaces) format
	bindings := make([]string, 0, len(up.Status.Bindings))

	for _, binding := range up.Status.Bindings {
		switch binding.Scope {
		case clusterviewv1alpha1.BindingScopeCluster:
			// Cluster-scoped: cluster(*)
			bindings = append(bindings, fmt.Sprintf("%s(*)", binding.Cluster))
		case clusterviewv1alpha1.BindingScopeNamespace:
			// Namespace-scoped: cluster(ns1,ns2,...)
			namespaces := make([]string, 0)
			for _, ns := range binding.Namespaces {
				if ns != "*" {
					namespaces = append(namespaces, ns)
				}
			}
			if len(namespaces) > 0 {
				bindings = append(bindings, fmt.Sprintf("%s(%s)", binding.Cluster, strings.Join(namespaces, ",")))
			}
		}
	}

	bindingsDisplay := formatBindings(bindings)

	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: up},
	}
	row.Cells = append(row.Cells,
		up.Name,
		bindingsDisplay,
	)

	return row
}

// formatBindings formats bindings for display
func formatBindings(bindings []string) string {
	if len(bindings) == 0 {
		return "-"
	}
	// Join all bindings with space separator for readability
	return strings.Join(bindings, " ")
}

var _ = rest.Getter(&REST{})

// Get retrieves a specific UserPermission by ClusterRole name
func (s *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(clusterviewv1alpha1.Resource("userpermissions"), "", fmt.Errorf("unable to get userpermission without a user on the context"))
	}

	return s.cache.Get(user, name)
}

var _ = rest.SingularNameProvider(&REST{})

func (s *REST) GetSingularName() string {
	return "userpermission"
}
