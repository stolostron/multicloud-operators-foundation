package clusterpool

import (
	"context"

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
	clusterPool     = "clusterpools"
)

// This controller maintain the clusterset to clusterpool map. the map value of clusterpool is clusterpools/<Namespace>/<Name>
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
	c, err := controller.New("clusterset-clusterpool-mapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &hivev1.ClusterPool{}},
		&handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// put all clusterpool which have this clusterset label into queue if there is the clusterset event
	err = c.Watch(&source.Kind{Type: &clusterv1alapha1.ManagedClusterSet{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
				if _, ok := obj.Object.(*clusterv1alapha1.ManagedClusterSet); !ok {
					// not a clusterset, returning empty
					klog.Error("clusterset handler received non-clusterset object")
					return []reconcile.Request{}
				}

				clusterpoolList := &hivev1.ClusterPoolList{}

				//List Clusterset related cluster
				labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
					ClusterSetLabel: obj.Meta.GetName(),
				}}
				selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
				if err != nil {
					return nil
				}

				err = mgr.GetClient().List(context.TODO(), clusterpoolList, &client.ListOptions{LabelSelector: selector})
				if err != nil {
					klog.Errorf("failed to list clusterpool %v", err)
				}

				var requests []reconcile.Request
				for _, clusterpool := range clusterpoolList.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: clusterpool.Namespace,
							Name:      clusterpool.Name,
						},
					})
				}

				klog.V(5).Infof("List clusterpool %+v", requests)
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
	clusterpool := &hivev1.ClusterPool{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, req.NamespacedName, clusterpool)
	if err != nil {
		if errors.IsNotFound(err) {
			r.clusterSetMapper.DeleteObjectInClusterSet(utils.ResourceNamespacedName(clusterPool, req.Namespace, req.Name))
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if _, ok := clusterpool.Labels[ClusterSetLabel]; !ok {
		r.clusterSetMapper.DeleteObjectInClusterSet(utils.ResourceNamespacedName(clusterPool, req.Namespace, req.Name))
		return ctrl.Result{}, nil
	}

	clustersetName := clusterpool.Labels[ClusterSetLabel]

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
	r.clusterSetMapper.UpdateObjectInClusterSet(utils.ResourceNamespacedName(clusterPool, req.Namespace, req.Name), clustersetName)
	klog.V(5).Infof("clusterSetMapper: %+v", r.clusterSetMapper.GetAllClusterSetToObjects())
	return ctrl.Result{}, nil
}
