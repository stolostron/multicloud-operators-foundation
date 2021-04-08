package clusterclaim

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
	ClusterSetLabel = "cluster.open-cluster-management.io/clusterset"
	clusterClaim    = "clusterclaims"
)

// This controller maintain the clusterset to clusterclaims map.  the the map value of clustercalaim is clusterclaims/<Namespace>/<Name>
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
	c, err := controller.New("clusterset-clusterclaim-mapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &hivev1.ClusterClaim{}},
		&handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// put all clusterclaim which have this clusterset label into queue if there is the clusterset event
	err = c.Watch(&source.Kind{Type: &clusterv1alapha1.ManagedClusterSet{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
				if _, ok := obj.Object.(*clusterv1alapha1.ManagedClusterSet); !ok {
					// not a clusterset, returning empty
					klog.Error("clusterset handler received non-clusterset object")
					return []reconcile.Request{}
				}

				clusterclaimList := &hivev1.ClusterClaimList{}

				//List Clusterset related cluster
				labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
					ClusterSetLabel: obj.Meta.GetName(),
				}}
				selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
				if err != nil {
					return nil
				}

				err = mgr.GetClient().List(context.TODO(), clusterclaimList, &client.ListOptions{LabelSelector: selector})
				if err != nil {
					klog.Errorf("failed to list clusterclaim %v", err)
				}

				var requests []reconcile.Request
				for _, clusterclaim := range clusterclaimList.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: clusterclaim.Namespace,
							Name:      clusterclaim.Name,
						},
					})
				}

				klog.V(5).Infof("List clusterclaim %+v", requests)
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
	clusterclaim := &hivev1.ClusterClaim{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, req.NamespacedName, clusterclaim)
	if err != nil {
		if errors.IsNotFound(err) {
			r.clusterSetMapper.DeleteObjectInClusterSet(utils.ResourceNamespacedName(clusterClaim, req.Namespace, req.Name))
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//update clusterclaim label by clusterpool's clusterset label
	clusterpool := &hivev1.ClusterPool{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: clusterclaim.Namespace, Name: clusterclaim.Spec.ClusterPoolName}, clusterpool)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(3).Infof("Can not get clusterclaim's clusterpool: %+v", clusterclaim.Spec.ClusterPoolName)
			// If clusterclaim's clusterpool clusterset label has been deleted. should also remove clusterset label for clusterclaim.
			if _, ok := clusterclaim.Labels[ClusterSetLabel]; ok {
				delete(clusterclaim.Labels, ClusterSetLabel)
				err = r.client.Update(ctx, clusterclaim, &client.UpdateOptions{})
				if err != nil {
					klog.Errorf("Can not update clusterclaim label: %+v", clusterclaim.Name)
					return ctrl.Result{}, err
				}
			}
			r.clusterSetMapper.DeleteObjectInClusterSet(utils.ResourceNamespacedName(clusterClaim, req.Namespace, req.Name))
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	klog.V(5).Infof("Clusterclaim's clusterpool: %+v", clusterpool)
	oriLabel := clusterclaim.Labels
	if clusterpool.Labels != nil && len(clusterpool.Labels[ClusterSetLabel]) != 0 {
		//if clusterpool has clusterset label, add clusterset label to clusterclaim.
		if clusterclaim.Labels == nil {
			clusterclaim.Labels = make(map[string]string)
		}
		clusterclaim.Labels[ClusterSetLabel] = clusterpool.Labels[ClusterSetLabel]
	} else {
		//if clusterpool do not has clusterset label, clean clusterset label from clusterclaim.
		if clusterclaim.Labels != nil && len(clusterclaim.Labels[ClusterSetLabel]) != 0 {
			delete(clusterclaim.Labels, ClusterSetLabel)
		}
	}

	if !reflect.DeepEqual(oriLabel, clusterclaim.Labels) {
		err = r.client.Update(ctx, clusterclaim, &client.UpdateOptions{})
		if err != nil {
			klog.Errorf("Can not update clusterclaim label: %+v", clusterclaim.Name)
			return ctrl.Result{}, err
		}
	}

	if len(clusterclaim.Labels[ClusterSetLabel]) == 0 {
		r.clusterSetMapper.DeleteObjectInClusterSet(utils.ResourceNamespacedName(clusterClaim, req.Namespace, req.Name))
		return ctrl.Result{}, nil
	}

	clustersetName := clusterclaim.Labels[ClusterSetLabel]

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

	//update clusterclaim in map.
	r.clusterSetMapper.UpdateObjectInClusterSet(utils.ResourceNamespacedName(clusterClaim, req.Namespace, req.Name), clustersetName)
	klog.V(5).Infof("clusterSetMapper: %+v", r.clusterSetMapper.GetAllClusterSetToObjects())
	return ctrl.Result{}, nil
}
