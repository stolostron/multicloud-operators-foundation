package clusterclaim

import (
	"context"
	"fmt"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	clusterclientset "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterv1alpha1informer "github.com/open-cluster-management/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type ListClusterClaimsFunc func() ([]*clusterv1alpha1.ClusterClaim, error)

// ClusterClaimReconciler reconciles cluster claim objects
type ClusterClaimReconciler struct {
	Log               logr.Logger
	ListClusterClaims ListClusterClaimsFunc
	ClusterClient     clusterclientset.Interface
	ClusterInformers  clusterv1alpha1informer.ClusterClaimInformer
}

func (r *ClusterClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := r.syncClaims(ctx)
	return ctrl.Result{}, err
}

func (r *ClusterClaimReconciler) syncClaims(ctx context.Context) error {
	r.Log.V(4).Info("Sync cluster claims")
	claims, err := r.ListClusterClaims()
	if err != nil {
		return err
	}

	errs := []error{}
	claimSet := sets.String{}

	for _, claim := range claims {
		if err := r.createOrUpdate(ctx, claim); err != nil {
			errs = append(errs, err)
		}
		claimSet.Insert(claim.Name)
	}

	existedObjs, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().List(ctx, metav1.ListOptions{})
	if err != nil {
		errs = append(errs, client.IgnoreNotFound(err))
		return utils.NewMultiLineAggregate(errs)
	}

	for _, claim := range existedObjs.Items {
		if claimSet.Has(claim.Name) {
			continue
		}
		err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Delete(ctx, claim.Name, metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utils.NewMultiLineAggregate(errs)
}

func (r *ClusterClaimReconciler) createOrUpdate(ctx context.Context, newClaim *clusterv1alpha1.ClusterClaim) error {
	oldClaim, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Get(ctx, newClaim.Name, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err):
		_, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Create(ctx, newClaim, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("unable to create ClusterClaim: %v, %w", newClaim, err)
		}
	case err != nil:
		return fmt.Errorf("unable to get ClusterClaim %q: %w", newClaim.Name, err)
	case !reflect.DeepEqual(oldClaim.Spec, newClaim.Spec):
		oldClaim.Spec = newClaim.Spec
		_, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Update(ctx, oldClaim, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update ClusterClaim %q: %w", oldClaim.Name, err)
		}
	}
	return nil
}

func (r *ClusterClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// setup a controller for ClusterClaim with customized event source and handler
	c, err := controller.New("ClusterClaim", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	claimSource := NewClusterClaimSource(r.ClusterInformers)
	if err := c.Watch(claimSource, &ClusterClaimEventHandler{}); err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &clusterv1beta1.ManagedClusterInfo{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

// NewClusterClaimSource returns an event source for cluster claims
func NewClusterClaimSource(clusterInformers clusterv1alpha1informer.ClusterClaimInformer) source.Source {
	return &clusterClaimSource{
		claimInformer: clusterInformers.Informer(),
	}
}

// clusterClaimSource is the event source of cluster claims on managed cluster.
type clusterClaimSource struct {
	claimInformer cache.SharedIndexInformer
}

var _ source.SyncingSource = &clusterClaimSource{}

func (s *clusterClaimSource) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate) error {
	// all predicates are ignored
	s.claimInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			handler.Create(event.CreateEvent{}, queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			handler.Update(event.UpdateEvent{}, queue)
		},
		DeleteFunc: func(obj interface{}) {
			handler.Delete(event.DeleteEvent{}, queue)
		},
	})

	return nil
}

func (s *clusterClaimSource) WaitForSync(ctx context.Context) error {
	if ok := cache.WaitForCacheSync(ctx.Done(), s.claimInformer.HasSynced); !ok {
		return fmt.Errorf("Never achieved initial sync")
	}
	return nil
}

// ClusterClaimEventHandler maps any event to an empty request
type ClusterClaimEventHandler struct{}

var _ handler.EventHandler = &ClusterClaimEventHandler{}

func (e *ClusterClaimEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *ClusterClaimEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *ClusterClaimEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *ClusterClaimEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}
