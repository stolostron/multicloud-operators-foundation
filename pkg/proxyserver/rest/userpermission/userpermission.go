package userpermission

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

type REST struct {
	cache          cache.UserPermissionLister
	tableConverter rest.TableConvertor
}

// NewREST returns a RESTStorage object that will work against UserPermission resources
func NewREST(cache cache.UserPermissionLister) *REST {
	return &REST{
		cache:          cache,
		tableConverter: rest.NewDefaultTableConvertor(clusterviewv1alpha1.Resource("userpermissions")),
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

func (c *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return c.tableConverter.ConvertToTable(ctx, object, tableOptions)
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
