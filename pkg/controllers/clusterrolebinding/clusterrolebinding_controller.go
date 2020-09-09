package clusterrolebinding

import (
	"context"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ClustersetFinalizerName string = "open-cluster-management.io/clusterset"
	ClusterSetLabel         string = "clusterset"
)

//This Controller will generate a Clusterset to Subjects map, and this map will be used to sync
// clusterset related clusterrolebinding.
type Reconciler struct {
	client                  client.Client
	scheme                  *runtime.Scheme
	clusterroleToClusterset map[string]sets.String
	clustersetToSubject     *helpers.ClustersetSubjectsMapper
}

func SetupWithManager(mgr manager.Manager, clustersetToSubject *helpers.ClustersetSubjectsMapper) error {
	if err := add(mgr, newReconciler(mgr, clustersetToSubject)); err != nil {
		klog.Errorf("Failed to create auto-detect controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, clustersetToSubject *helpers.ClustersetSubjectsMapper) reconcile.Reconciler {
	var clusterroleToClusterset = make(map[string]sets.String)
	return &Reconciler{
		client:                  mgr.GetClient(),
		scheme:                  mgr.GetScheme(),
		clustersetToSubject:     clustersetToSubject,
		clusterroleToClusterset: clusterroleToClusterset,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterrolebinding-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &rbacv1.ClusterRoleBinding{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
				clusterrolebinding, ok := a.Object.(*rbacv1.ClusterRoleBinding)
				if !ok {
					// not a clusterrolebinding, returning empty
					klog.Error("Clusterrolebinding handler received non-SyncSet object")
					return []reconcile.Request{}
				}
				var requests []reconcile.Request
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: clusterrolebinding.RoleRef.Name,
					},
				})
				return requests
			}),
		})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	clusterrole := &rbacv1.ClusterRole{}
	err := r.client.Get(ctx, req.NamespacedName, clusterrole)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	clustersetNames := sets.NewString()
	//generate clusterrole's all clustersets
	curClustersetInRule := utils.GetClustersetInRules(clusterrole.Rules)
	clustersetNames.Insert(curClustersetInRule...)

	//Both current clustersets and previous clustrersets for this clusterrole
	unionClustersetNames := clustersetNames.Union(r.clusterroleToClusterset[clusterrole.Name])

	//Add Finalizer to clusterset related clusterrole
	if clustersetNames.Len() != 0 && !utils.ContainsString(clusterrole.GetFinalizers(), ClustersetFinalizerName) {
		klog.Infof("adding ClusterRoleBinding Finalizer to ClusterRole %v", clusterrole.Name)
		clusterrole.ObjectMeta.Finalizers = append(clusterrole.ObjectMeta.Finalizers, ClustersetFinalizerName)
		if err := r.client.Update(context.TODO(), clusterrole); err != nil {
			klog.Warningf("will reconcile since failed to add finalizer to ClusterRole %v, %v", clusterrole.Name, err)
			return reconcile.Result{}, err
		}
	}

	//clusterrole is deleted or clusterrole is not related to any clusterset
	if clustersetNames.Len() == 0 || !clusterrole.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(clusterrole.GetFinalizers(), ClustersetFinalizerName) {
			klog.Infof("removing ClusterRoleBinding Finalizer in ClusterRole %v", clusterrole.Name)
			clusterrole.ObjectMeta.Finalizers = utils.RemoveString(clusterrole.ObjectMeta.Finalizers, ClustersetFinalizerName)
			if err := r.client.Update(context.TODO(), clusterrole); err != nil {
				klog.Warningf("will reconcile since failed to remove Finalizer from ClusterRole %v, %v", clusterrole.Name, err)
				return reconcile.Result{}, err
			}
		}
		if _, ok := r.clusterroleToClusterset[clusterrole.Name]; !ok {
			return ctrl.Result{}, nil
		}
		delete(r.clusterroleToClusterset, clusterrole.Name)
	} else {
		r.clusterroleToClusterset[clusterrole.Name] = clustersetNames
	}

	curClustersetToRoles := generateClustersetToClusterroles(r.clusterroleToClusterset)

	for curClusterset := range unionClustersetNames {
		var clustersetSubjects []rbacv1.Subject
		for _, curClusterRoleName := range curClustersetToRoles[curClusterset] {
			subjects, err := r.getClusterroleSubject(ctx, curClusterRoleName)
			if err != nil {
				return ctrl.Result{}, err
			}
			clustersetSubjects = utils.Mergesubjects(clustersetSubjects, subjects)
		}
		r.clustersetToSubject.Set(curClusterset, clustersetSubjects)
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) getClusterroleSubject(ctx context.Context, clusterroleName string) ([]rbacv1.Subject, error) {
	var subjects []rbacv1.Subject

	clusterrolebindinglist := &rbacv1.ClusterRoleBindingList{}
	err := r.client.List(ctx, clusterrolebindinglist)
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	for _, clusterrolebinding := range clusterrolebindinglist.Items {
		if _, ok := clusterrolebinding.Labels[ClusterSetLabel]; ok {
			continue
		}
		if clusterrolebinding.RoleRef.APIGroup == rbacv1.GroupName && clusterrolebinding.RoleRef.Kind == "ClusterRole" && clusterrolebinding.RoleRef.Name == clusterroleName {
			subjects = utils.Mergesubjects(subjects, clusterrolebinding.Subjects)

		}
	}
	return subjects, nil
}

func generateClustersetToClusterroles(clusterroleToClusterset map[string]sets.String) map[string][]string {
	var clustersetToClusterroles = make(map[string][]string)
	for currole, cursets := range clusterroleToClusterset {
		for curset := range cursets {
			clustersetToClusterroles[curset] = append(clustersetToClusterroles[curset], currole)
		}
	}
	return clustersetToClusterroles
}
