package project

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog"

	projectv1 "github.com/openshift/api/project/v1"

	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/printers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/printers/storage"
)

type REST struct {
	projectCache   *cache.KubevirtProjectCache
	tableConverter rest.TableConvertor
}

var (
	_ = rest.Scoper(&REST{})
	_ = rest.KindProvider(&REST{})
	_ = rest.SingularNameProvider(&REST{})
	_ = rest.Storage(&REST{})
	_ = rest.Lister(&REST{})
	_ = rest.Getter(&REST{})
)

func (r *REST) NamespaceScoped() bool {
	return false
}

func (s *REST) Kind() string {
	return "Projects"
}

func (r *REST) GetSingularName() string {
	return "project"
}

// NewREST returns a RESTStorage for projects based on ClusterPermission
func NewREST(projectCache *cache.KubevirtProjectCache) *REST {
	return &REST{
		projectCache: projectCache,
		// tableConverter: rest.NewDefaultTableConvertor(projectv1.Resource("projects")),
		tableConverter: storage.TableConvertor{TableGenerator: printers.NewTableGenerator().With(func(ph printers.PrintHandler) {
			columnDefinitions := []metav1.TableColumnDefinition{
				{Name: "Name", Type: "string", Description: "The name of project."},
				{Name: "Cluster", Type: "string", Description: "The managed cluster of project."},
			}
			err := ph.TableHandler(columnDefinitions, printProject)
			if err != nil {
				klog.Warningf("%v", err)
			}
			err = ph.TableHandler(columnDefinitions, printProjectList)
			if err != nil {
				klog.Warningf("%v", err)
			}
		})},
	}
}

func (r *REST) New() runtime.Object {
	return &projectv1.Project{}
}

func (r *REST) Destroy() {
}

func (r *REST) NewList() runtime.Object {
	return &projectv1.ProjectList{}
}

func (r *REST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(projectv1.Resource("projects"), "", fmt.Errorf("unable to list projects without a user on the context"))
	}

	labelSelector, _ := helpers.InternalListOptionsToSelectors(options)
	projectList, err := r.projectCache.List(user, labelSelector)
	if err != nil {
		return nil, err
	}

	return projectList, nil
}

func (r *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return r.tableConverter.ConvertToTable(ctx, object, tableOptions)
}

func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(projectv1.Resource("projects"), "", fmt.Errorf("unable to get projects without a user on the context"))
	}

	projectList, err := r.projectCache.List(user, labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, project := range projectList.Items {
		if name == project.Name {
			return &project, nil
		}
	}

	return nil, errors.NewForbidden(projectv1.Resource("projects"), "", fmt.Errorf("the user cannot get the projects %v", name))
}

func printProject(obj *projectv1.Project, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}

	cluster, ok := obj.Labels[cache.KubeVirtProjectClusterLabel]
	if !ok {
		return nil, fmt.Errorf("failed to get cluster for project %s", obj.Name)
	}

	row.Cells = append(row.Cells, obj.Name, cluster)
	return []metav1.TableRow{row}, nil
}

func printProjectList(list *projectv1.ProjectList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printProject(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}
