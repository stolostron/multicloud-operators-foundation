package clusterdeployment

import (
	"context"
	"reflect"

	clusterv1alapha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const (
	ClusterSetLabel   = "cluster.open-cluster-management.io/clusterset"
	clusterDeployment = "clusterdeployments"
)

// This controller maintain the clusterset to clusterdeployment map. the the map value of clusterdeployment is clusterdeployments/<Namespace>/<Name>
type Reconciler struct {
	client           client.Client
	scheme           *runtime.Scheme
	clusterSetMapper *helpers.ClusterSetMapper
}

func SetupWithManager(mgr manager.Manager, clusterSetMapper *helpers.ClusterSetMapper) error {
	if err := add(mgr, newReconciler(mgr, clusterSetMapper)); err != nil {
		klog.Errorf("Failed to create ClusterSetMapper controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, clusterSetMapper *helpers.ClusterSetMapper) reconcile.Reconciler {
	return &Reconciler{
		client:           mgr.GetClient(),
		scheme:           mgr.GetScheme(),
		clusterSetMapper: clusterSetMapper,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterset-clusterdeployment-mapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &hivev1.ClusterDeployment{}},
		&handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// put all clusterdeployment which have this clusterset label into queue if there is the clusterset event
	err = c.Watch(&source.Kind{Type: &clusterv1alapha1.ManagedClusterSet{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
				if _, ok := obj.Object.(*clusterv1alapha1.ManagedClusterSet); !ok {
					// not a clusterset, returning empty
					klog.Error("clusterset handler received non-clusterset object")
					return []reconcile.Request{}
				}

				clusterdeploymentList := &hivev1.ClusterDeploymentList{}

				//List Clusterset related cluster
				labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
					ClusterSetLabel: obj.Meta.GetName(),
				}}
				selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
				if err != nil {
					return nil
				}

				err = mgr.GetClient().List(context.TODO(), clusterdeploymentList, &client.ListOptions{LabelSelector: selector})
				if err != nil {
					klog.Errorf("failed to list clusterdeploymentSet %v", err)
				}

				var requests []reconcile.Request
				for _, clusterdeployment := range clusterdeploymentList.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: clusterdeployment.Namespace,
							Name:      clusterdeployment.Name,
						},
					})
				}

				klog.V(5).Infof("List clusterdeployment %+v", requests)
				return requests
			}),
		})
	if err != nil {
		return nil
	}
	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	clusterdeployment := &hivev1.ClusterDeployment{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, req.NamespacedName, clusterdeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			// clusterdeployment has been deleted
			r.clusterSetMapper.DeleteObjectInClusterSet(utils.ResourceNamespacedName(clusterDeployment, req.Namespace, req.Name))
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//update clusterdeployment label by clusterpool's clusterset label if this clusterdeployment is created using clusterpool
	if clusterdeployment.Spec.ClusterPoolRef != nil {
		clusterpool := &hivev1.ClusterPool{}
		err := r.client.Get(ctx, types.NamespacedName{Namespace: clusterdeployment.Spec.ClusterPoolRef.Namespace, Name: clusterdeployment.Spec.ClusterPoolRef.PoolName}, clusterpool)
		if err != nil {
			if errors.IsNotFound(err) {
				klog.V(3).Infof("Can not get clusterdeployment's clusterpool: %+v", clusterdeployment.Spec.ClusterPoolRef.PoolName)
				// If clusterdeployment's clusterpool clusterset label has been deleted. should also remove clusterset label for clusterdeployment.
				if _, ok := clusterdeployment.Labels[ClusterSetLabel]; ok {
					delete(clusterdeployment.Labels, ClusterSetLabel)
					err = r.client.Update(ctx, clusterdeployment, &client.UpdateOptions{})
					if err != nil {
						klog.Errorf("Can not update clusterdeployment label: %+v", clusterdeployment.Name)
						return ctrl.Result{}, err
					}
				}
				r.clusterSetMapper.DeleteObjectInClusterSet(utils.ResourceNamespacedName(clusterDeployment, req.Namespace, req.Name))
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		klog.V(5).Infof("Clusterdeployment's clusterpool: %+v", clusterpool)
		oriLabel := clusterdeployment.Labels

		if clusterpool.Labels != nil && len(clusterpool.Labels[ClusterSetLabel]) != 0 {
			//if clusterpool has clusterset label, add clusterset label to clusterdeployment.
			if clusterdeployment.Labels == nil {
				clusterdeployment.Labels = make(map[string]string)
			}
			clusterdeployment.Labels[ClusterSetLabel] = clusterpool.Labels[ClusterSetLabel]
		} else {
			//if clusterpool do not has clusterset label, clean clusterset label from clusterdeployment.
			if clusterdeployment.Labels != nil && len(clusterdeployment.Labels[ClusterSetLabel]) != 0 {
				delete(clusterdeployment.Labels, ClusterSetLabel)
			}
		}

		if !reflect.DeepEqual(oriLabel, clusterdeployment.Labels) {
			err = r.client.Update(ctx, clusterdeployment, &client.UpdateOptions{})
			if err != nil {
				klog.Errorf("Can not update clusterdeployment label: %+v", clusterdeployment.Name)
				return ctrl.Result{}, err
			}
		}
	}

	if len(clusterdeployment.Labels[ClusterSetLabel]) == 0 {
		r.clusterSetMapper.DeleteObjectInClusterSet(utils.ResourceNamespacedName(clusterDeployment, req.Namespace, req.Name))
		return ctrl.Result{}, nil
	}

	clustersetName := clusterdeployment.Labels[ClusterSetLabel]

	//If the managedclusterset do not exist, delete this clusterset in map
	clusterset := &clusterv1alapha1.ManagedClusterSet{}
	err = r.client.Get(ctx, types.NamespacedName{Name: clustersetName}, clusterset)
	if err != nil {
		if errors.IsNotFound(err) {
			r.clusterSetMapper.DeleteClusterSet(clustersetName)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.clusterSetMapper.UpdateObjectInClusterSet(utils.ResourceNamespacedName(clusterDeployment, req.Namespace, req.Name), clustersetName)
	klog.V(5).Infof("clusterSetMapper: %+v", r.clusterSetMapper.GetAllClusterSetToObjects())
	return ctrl.Result{}, nil
}
