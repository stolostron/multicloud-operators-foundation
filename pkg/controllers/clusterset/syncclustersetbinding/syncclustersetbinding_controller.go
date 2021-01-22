package syncclustersetbinding

import (
	"context"
	"time"

	v1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ServiceAccountName string = "default"
)

//This controller apply clusterset related clusterrolebinding based on clustersetToClusters and clustersetToSubject map
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create ClustersetBinding controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterinfo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//reconcile every 10s
	reconcile := reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}

	ctx := context.Background()
	nsToClustersets := make(map[string]sets.String)
	// List clusterrolebinding
	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
	err := r.client.List(ctx, clusterRoleBindingList, &client.ListOptions{})
	if err != nil {
		return reconcile, err
	}
	for _, curClusterRoleBinding := range clusterRoleBindingList.Items {
		nsList := getServiceAccountNamespaces(&curClusterRoleBinding)
		if nsList.Len() == 0 {
			continue
		}
		//klog.Errorf("####nslits: %v", nsList)
		//Get clusterrole
		clusterRole := &rbacv1.ClusterRole{}
		err := r.client.Get(ctx, types.NamespacedName{Name: curClusterRoleBinding.RoleRef.Name}, clusterRole)
		if err != nil {
			//	klog.Errorf("Failed to get clusterRole, %v", err)
			continue
		}
		//klog.Errorf("####clusterrole: %v", clusterRole)

		clustersetsInRule := utils.GetClustersetInRules(clusterRole.Rules)
		if clusterRole.Name == "clustersetrole1" {
			klog.Errorf("####clusterrole: %v", clusterRole)
			klog.Errorf("####clustersetsInRule: %v", clustersetsInRule)
		}

		if len(clustersetsInRule) == 0 {
			continue
		}
		// klog.Errorf("####clusterrolebinding: %v", curClusterRoleBinding)
		// klog.Errorf("####clustersetsrole: %v", clustersetsInRule)
		// klog.Errorf("####clusterRole.Rules: %v", clusterRole.Rules)
		// klog.Errorf("####clusterRole: %v", clusterRole)

		//ns->clusterset

		for ns := range nsList {
			if clustersetsInRule.Has("*") {
				nsToClustersets[ns] = sets.NewString("*")
				continue
			}

			nsToClustersets[ns] = nsToClustersets[ns].Union(clustersetsInRule)
		}

	}
	klog.Errorf("####nsToClustersets: %v", nsToClustersets)
	r.applyDesiredClustersetBinding(ctx, nsToClustersets)
	return reconcile, err
}

func getServiceAccountNamespaces(curClusterRoleBinding *rbacv1.ClusterRoleBinding) sets.String {
	nsSet := sets.NewString()
	subjects := curClusterRoleBinding.Subjects
	for _, subject := range subjects {
		if subject.Kind == "ServiceAccount" && subject.Name == ServiceAccountName {
			nsSet.Insert(subject.Namespace)
			klog.Errorf("####subject: %v", subject)
		}
	}
	return nsSet
}

func (r *Reconciler) applyDesiredClustersetBinding(ctx context.Context, nsToClustersets map[string]sets.String) {
	// klog.Errorf("####nsToClustersets: %v", nsToClustersets)
	clustersetBindingList := &v1alpha1.ManagedClusterSetBindingList{}
	err := r.client.List(ctx, clustersetBindingList, &client.ListOptions{})
	if err != nil {
		return
	}
	for _, curClustersetBinding := range clustersetBindingList.Items {
		if _, ok := nsToClustersets[curClustersetBinding.Namespace]; ok {
			continue
		}
		err := r.client.Delete(ctx, &curClustersetBinding, &client.DeleteOptions{})
		if err != nil {
			klog.Errorf("Failed to delete clustersetbinding: %v. error: %v", curClustersetBinding, err)
		}
	}

	for ns, clustersets := range nsToClustersets {
		desiredClusterset := sets.NewString()
		if clustersets.Has("*") {
			clustersetList := &v1alpha1.ManagedClusterSetList{}
			err := r.client.List(ctx, clustersetList, &client.ListOptions{})
			if err != nil {
				return
			}
			for _, indexClusterset := range clustersetList.Items {
				desiredClusterset.Insert(indexClusterset.Name)
			}
		} else {
			desiredClusterset = clustersets
		}
		// klog.Errorf("####ns: %v", ns)
		// klog.Errorf("####desiredClusterset: %v", desiredClusterset)
		//Get current clustersetbinding
		clustersetBindingList := &v1alpha1.ManagedClusterSetBindingList{}
		err := r.client.List(ctx, clustersetBindingList, &client.ListOptions{Namespace: ns})
		if err != nil {
			return
		}
		// klog.Errorf("####clustersetBindingList.Items: %v", clustersetBindingList.Items)
		for _, clustersetbinding := range clustersetBindingList.Items {
			// klog.Errorf("####clustersetbinding: %v", clustersetbinding)
			if !desiredClusterset.Has(clustersetbinding.Name) {
				// klog.Errorf("####clustersetbinding.Name: %v", clustersetbinding.Name)
				err := r.client.Delete(ctx, &clustersetbinding, &client.DeleteOptions{})
				if err != nil {
					klog.Errorf("Failed to delete clustersetbinding: %v. error: %v", clustersetbinding, err)
				}
			} else {
				desiredClusterset.Delete(clustersetbinding.Name)
				// klog.Errorf("####clustersetbinding.Name: %v", clustersetbinding.Name)
			}
		}
		// klog.Errorf("####desiredClusterset: %v", desiredClusterset)
		for clusterset := range desiredClusterset {
			newClustersetBinding := &v1alpha1.ManagedClusterSetBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterset,
					Namespace: ns,
				},
				Spec: v1alpha1.ManagedClusterSetBindingSpec{
					ClusterSet: clusterset,
				},
			}
			err := r.client.Create(ctx, newClustersetBinding, &client.CreateOptions{})
			if err != nil {
				klog.Errorf("failed to create clustersetbinding: %v error: %v", newClustersetBinding, err)
			}
		}
	}
	return
}

