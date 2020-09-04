package clusterrolebinding

import (
	"context"

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

var ClusterSetLabel string = "Clustersetlabel"

type Reconciler struct {
	client                  client.Client
	scheme                  *runtime.Scheme
	clusterroleToClusterset map[string]sets.String
	clustersetToSubject     map[string][]rbacv1.Subject
}

func SetupWithManager(mgr manager.Manager, clustersetToSubject map[string][]rbacv1.Subject) error {

	if err := add(mgr, newReconciler(mgr, clustersetToSubject)); err != nil {
		klog.Errorf("Failed to create auto-detect controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, clustersetToSubject map[string][]rbacv1.Subject) reconcile.Reconciler {
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
	//	var clusterroleToClusterset map[string][]string
	ctx := context.Background()
	clusterrole := &rbacv1.ClusterRole{}
	klog.Errorf("#########In reconcile0:req.NamespacedName:%+v", req.NamespacedName)
	klog.Errorf("#########In reconcile end0:  r.clusterroleToClusterset:%v", r.clusterroleToClusterset)
	klog.Errorf("#########In reconcile end1:clustersetToSubject: %v", r.clustersetToSubject)

	//TODO  deleted ,cannot get
	err := r.client.Get(ctx, req.NamespacedName, clusterrole)
	if err != nil {
		klog.Errorf("#########error reconcile1:,error:%+v", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	clustersetNames := sets.NewString()
	//generate clusterrole to clusterset map
	curClustersetInRule := r.getClustersetInRules(clusterrole.Rules)
	clustersetNames.Insert(curClustersetInRule...)

	if clustersetNames.Len() == 0 || !clusterrole.GetDeletionTimestamp().IsZero() {
		if _, ok := r.clusterroleToClusterset[clusterrole.Name]; !ok {
			return ctrl.Result{}, nil
		}
	}
	if clustersetNames.Equal(r.clusterroleToClusterset[clusterrole.Name]) {
		return ctrl.Result{}, nil
	}

	//Changed clusterrole related clusterset, should recaculate all this clusterrole related clusterset's subjects
	//Both current clusterset names and previous clusterset names
	unionClustersetNames := clustersetNames.Union(r.clusterroleToClusterset[clusterrole.Name])

	unionClusterroleToClusterset := r.clusterroleToClusterset[clusterrole.Name].Union(clustersetNames)
	r.clusterroleToClusterset[clusterrole.Name] = unionClusterroleToClusterset
	clustersetToClusterroles := r.generateClustersetToClusterroles()

	klog.Errorf("######unionClustersetNames:%+v", unionClustersetNames)

	for curClusterset := range unionClustersetNames {
		klog.Errorf("######clustersetToClusterroles:%+v", clustersetToClusterroles)
		for _, curClusterRoleName := range clustersetToClusterroles[curClusterset] {
			subjects, err := r.getClusterroleSubject(ctx, curClusterRoleName)
			if err != nil {
				return ctrl.Result{}, err
			}
			r.clustersetToSubject[curClusterset] = r.mergesubjects(r.clustersetToSubject[curClusterset], subjects)
		}
	}

	//set clusterrole to clusterset map
	if len(curClustersetInRule) == 0 {
		delete(r.clusterroleToClusterset, clusterrole.Name)
	} else {
		r.clusterroleToClusterset[clusterrole.Name] = clustersetNames
	}

	klog.Errorf("#########In reconcile end0:  r.clusterroleToClusterset:%v", r.clusterroleToClusterset)
	klog.Errorf("#########In reconcile end1:clustersetToSubject: %v", r.clustersetToSubject)

	return ctrl.Result{}, nil
}

func (r *Reconciler) generateClustersetToClusterroles() map[string][]string {
	var clustersetToClusterroles = make(map[string][]string)
	for currole, cursets := range r.clusterroleToClusterset {
		for curset := range cursets {
			clustersetToClusterroles[curset] = append(clustersetToClusterroles[curset], currole)
		}
	}
	return clustersetToClusterroles
}

//get clusterrole's subject
func (r *Reconciler) getClusterroleSubject(ctx context.Context, clusterroleName string) ([]rbacv1.Subject, error) {
	var subjects []rbacv1.Subject

	//TODO should not get managedclusters rolebinding by ourselves
	clusterrolebindinglist := &rbacv1.ClusterRoleBindingList{}
	err := r.client.List(ctx, clusterrolebindinglist)
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	for _, clusterrolebinding := range clusterrolebindinglist.Items {
		if _, ok := clusterrolebinding.Labels[ClusterSetLabel]; ok {
			continue
		}
		if clusterrolebinding.RoleRef.APIGroup == "rbac.authorization.k8s.io" && clusterrolebinding.RoleRef.Kind == "ClusterRole" && clusterrolebinding.RoleRef.Name == clusterroleName {
			subjects = r.mergesubjects(subjects, clusterrolebinding.Subjects)

		}
	}
	return subjects, nil
}

func (r *Reconciler) mergesubjects(subjects []rbacv1.Subject, cursubjects []rbacv1.Subject) []rbacv1.Subject {
	var subjectmap = make(map[string]bool)
	returnSubjects := subjects
	for _, subject := range subjects {
		subkey := generateMapKey(subject)
		subjectmap[subkey] = true
	}
	for _, cursubject := range cursubjects {
		subkey := generateMapKey(cursubject)
		if _, ok := subjectmap[subkey]; !ok {
			returnSubjects = append(returnSubjects, cursubject)
		}
	}
	return returnSubjects
}

func generateMapKey(subject rbacv1.Subject) string {
	return subject.APIGroup + subject.Kind + subject.Name
}
func (r *Reconciler) getClustersetInRules(rules []rbacv1.PolicyRule) []string {
	var clustersetNames []string
	for _, rule := range rules {
		if IsContain(rule.APIGroups, "*") && IsContain(rule.Resources, "*") && IsContain(rule.Verbs, "*") {
			return []string{"*"}
		}
		if IsContain(rule.APIGroups, "cluster.open-cluster-management.io") {
			if IsContain(rule.Resources, "managedclustersets/bind") {
				if IsContain(rule.Verbs, "create") {
					for _, resourcename := range rule.ResourceNames {
						clustersetNames = append(clustersetNames, resourcename)
					}
				}
			}
		}
	}
	return clustersetNames
}

func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}
