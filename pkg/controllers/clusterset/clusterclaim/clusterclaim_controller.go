package clusterclaim

import (
	"context"

	utils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	clustersetutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/clusterset"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This controller sync the clusterclaim's utils.ClusterSetLabel with releated clusterpool's utils.ClusterSetLabel
// if the clusterpool did not exist, do nothing.
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create ClusterSetMapper controller, %v", err)
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
	c, err := controller.New("clusterset-clusterclaim-mapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &hivev1.ClusterClaim{}},
		&handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	//watch clusterdeployment related claim
	err = c.Watch(
		&source.Kind{Type: &hivev1.ClusterDeployment{}},
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(a client.Object) []reconcile.Request {
				clusterDeployment, ok := a.(*hivev1.ClusterDeployment)
				if !ok {
					// not a ClusterDeployment, returning empty
					klog.Error("ClusterDeployment handler received non-ClusterDeployment object")
					return []reconcile.Request{}
				}
				clusterclaims := &hivev1.ClusterClaimList{}
				err := mgr.GetClient().List(context.TODO(), clusterclaims, &client.ListOptions{})
				if err != nil {
					klog.Errorf("could not list clusterclaims. Error: %v", err)
				}
				var requests []reconcile.Request
				for _, clusterclaim := range clusterclaims.Items {
					//If clusterclaims is not related to this clusterDeployment, ignore it
					if clusterclaim.Spec.Namespace != clusterDeployment.Namespace {
						continue
					}
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      clusterclaim.Name,
							Namespace: clusterclaim.Namespace,
						},
					})
				}
				return requests
			}),
		))
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	clusterclaim := &hivev1.ClusterClaim{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, req.NamespacedName, clusterclaim)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//update clusterclaim label by clusterdeployment label
	if len(clusterclaim.Spec.Namespace) == 0 {
		return ctrl.Result{}, nil
	}
	clusterDeploymentName := clusterclaim.Spec.Namespace
	//update clusterclaim label by clusterpool's clusterset label
	clusterDeployment := &hivev1.ClusterDeployment{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: clusterDeploymentName, Name: clusterDeploymentName}, clusterDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	klog.V(5).Infof("Clusterclaim's clusterdeployment: %+v", clusterDeployment)

	var isModified = false
	utils.SyncMapFiled(&isModified, &clusterclaim.Labels, clusterDeployment.Labels, clustersetutils.ClusterSetLabel)

	if isModified {
		err = r.client.Update(ctx, clusterclaim, &client.UpdateOptions{})
		if err != nil {
			klog.Errorf("Can not update clusterclaim label: %+v", clusterclaim.Name)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
