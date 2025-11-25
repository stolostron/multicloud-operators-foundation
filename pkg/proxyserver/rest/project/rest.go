package project

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	cplisters "open-cluster-management.io/cluster-permission/client/listers/api/v1alpha1"

	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/printers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/printers/storage"
)

const (
	KubeVirtProjectClusterLabel = "cluster"
	KubeVirtProjectNameLabel    = "project"
	AllProjects                 = "all_projects"
)

type REST struct {
	indexer        cache.Indexer
	lister         cplisters.ClusterPermissionLister
	tableConverter rest.TableConvertor
}

var (
	_ = rest.Scoper(&REST{})
	_ = rest.KindProvider(&REST{})
	_ = rest.SingularNameProvider(&REST{})
	_ = rest.Storage(&REST{})
	_ = rest.Lister(&REST{})
)

func (r *REST) NamespaceScoped() bool {
	return false
}

func (s *REST) Kind() string {
	return "Project"
}

func (r *REST) GetSingularName() string {
	return "project"
}

// NewREST returns a RESTStorage for projects based on ClusterPermission
func NewREST(clusterPermissionIndexer cache.Indexer, clusterPermissionLister cplisters.ClusterPermissionLister) *REST {
	return &REST{
		indexer: clusterPermissionIndexer,
		lister:  clusterPermissionLister,
		tableConverter: storage.TableConvertor{TableGenerator: printers.NewTableGenerator().With(func(ph printers.PrintHandler) {
			columnDefinitions := []metav1.TableColumnDefinition{
				{Name: "Cluster", Type: "string", Description: "The managed cluster of project."},
				{Name: "Project", Type: "string", Description: "The name of project."},
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
	return &metav1.PartialObjectMetadata{}
}

func (r *REST) Destroy() {
}

func (r *REST) NewList() runtime.Object {
	return &metav1.PartialObjectMetadataList{}
}

func (r *REST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(schema.GroupResource{Resource: "projects"}, "", fmt.Errorf("unable to list projects without a user on the context"))
	}

	keys := r.indexer.ListIndexFuncValues(ClusterPermissionSubjectIndexKey)

	klog.V(4).Infof("list projects for user(groups=%v,name=%s) from %v", user.GetGroups(), user.GetName(), keys)

	projectSet := sets.New[projectView]()
	for _, key := range keys {
		namespace, name, subject, err := splitKey(key)
		if err != nil {
			return nil, err
		}

		if isBoundUser(subject, user) {
			obj, err := r.lister.ClusterPermissions(namespace).Get(name)
			if err != nil {
				return nil, err
			}

			//find the projects from ClusterPermission RoleBindings with the current user
			projectSet.Insert(listProjects(namespace, name, obj, user)...)
		}
	}

	projectList := projectSet.UnsortedList()
	sort.Slice(projectList, func(i, j int) bool {
		return projectList[i].cluster < projectList[j].cluster
	})

	projects := []metav1.PartialObjectMetadata{}
	for _, p := range projectList {
		projects = append(projects, metav1.PartialObjectMetadata{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "clusterview.open-cluster-management.io/v1",
				Kind:       "Project",
			},
			ObjectMeta: metav1.ObjectMeta{
				// using uuid (v5) generate an unique name for this resource based on cluster name and project name
				// the uuid (v5) will ensure a consistent id if using the same cluster name and project name, so that
				// if we want to retrieve this resource from the list, we use this name to find the corresponding resource
				Name: uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s-%s", p.cluster, p.name))).String(),
				Labels: map[string]string{
					KubeVirtProjectClusterLabel: p.cluster,
					KubeVirtProjectNameLabel:    p.name,
				},
			},
		})
	}

	return &metav1.PartialObjectMetadataList{Items: projects}, nil
}

func (r *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return r.tableConverter.ConvertToTable(ctx, object, tableOptions)
}

func printProject(obj *metav1.PartialObjectMetadata, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}

	cluster, ok := obj.Labels[KubeVirtProjectClusterLabel]
	if !ok {
		return nil, fmt.Errorf("failed to get cluster for project %s", obj.Name)
	}

	project, ok := obj.Labels[KubeVirtProjectNameLabel]
	if !ok {
		return nil, fmt.Errorf("failed to get project name for project %s", obj.Name)
	}
	if project == AllProjects {
		project = "*"
	}

	row.Cells = append(row.Cells, cluster, project)
	return []metav1.TableRow{row}, nil
}

func printProjectList(list *metav1.PartialObjectMetadataList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
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
