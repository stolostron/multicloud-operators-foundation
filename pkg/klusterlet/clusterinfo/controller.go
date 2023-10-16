package controllers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/agent"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	clusterv1alpha1informer "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	clusterv1alpha1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ClusterInfoReconciler reconciles a ManagedClusterInfo object
type ClusterInfoReconciler struct {
	client.Client
	Log                     logr.Logger
	Scheme                  *runtime.Scheme
	ManagedClusterClient    kubernetes.Interface
	ManagementClusterClient kubernetes.Interface
	NodeInformer            coreinformers.NodeInformer
	ClaimInformer           clusterv1alpha1informer.ClusterClaimInformer
	ClaimLister             clusterv1alpha1lister.ClusterClaimLister
	RouteV1Client           routev1.Interface
	ConfigV1Client          openshiftclientset.Interface
	ClusterName             string
	AgentName               string
	AgentNamespace          string
	AgentPort               int32
	Agent                   *agent.Agent
	// logging info syncer is used for search-ui only to get pod logs
	DisableLoggingInfoSyncer bool
}

type clusterInfoStatusSyncer interface {
	sync(ctx context.Context, clusterInfo *clusterv1beta1.ManagedClusterInfo) error
}

func (r *ClusterInfoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	request := types.NamespacedName{Namespace: r.ClusterName, Name: r.ClusterName}
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err := r.Get(ctx, request, clusterInfo)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if utils.ClusterIsOffLine(clusterInfo.Status.Conditions) {
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	newClusterInfo := clusterInfo.DeepCopy()

	syncers := []clusterInfoStatusSyncer{
		&defaultInfoSyncer{
			claimLister: r.ClaimLister,
		},
		&distributionInfoSyncer{
			configV1Client:       r.ConfigV1Client,
			managedClusterClient: r.ManagedClusterClient,
			claimLister:          r.ClaimLister,
		},
	}
	if !r.DisableLoggingInfoSyncer {
		syncers = append(syncers, &loggingInfoSyncer{
			clusterName:             r.ClusterName,
			agentName:               r.AgentName,
			agentNamespace:          r.AgentNamespace,
			agentPort:               r.AgentPort,
			agent:                   r.Agent,
			managementClusterClient: r.ManagementClusterClient,
			routeV1Client:           r.RouteV1Client,
			configV1Client:          r.ConfigV1Client,
		})
	}
	var errs []error
	for _, s := range syncers {
		if err := s.sync(ctx, newClusterInfo); err != nil {
			errs = append(errs, err)
		}
	}

	newSyncedCondition := metav1.Condition{
		Type:    clusterv1beta1.ManagedClusterInfoSynced,
		Status:  metav1.ConditionTrue,
		Reason:  clusterv1beta1.ReasonManagedClusterInfoSynced,
		Message: "Managed cluster info is synced",
	}
	if len(errs) > 0 {
		newSyncedCondition = metav1.Condition{
			Type:    clusterv1beta1.ManagedClusterInfoSynced,
			Status:  metav1.ConditionFalse,
			Reason:  clusterv1beta1.ReasonManagedClusterInfoSyncedFailed,
			Message: errors.NewAggregate(errs).Error(),
		}
	}
	meta.SetStatusCondition(&newClusterInfo.Status.Conditions, newSyncedCondition)

	// need to sync ocp ClusterVersion info every 5 min since do not watch it.
	if !clusterInfoStatusUpdated(&clusterInfo.Status, &newClusterInfo.Status) {
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	err = r.Client.Status().Update(ctx, newClusterInfo)
	if err != nil {
		klog.Error("Failed to update clusterInfo status. error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

func clusterInfoStatusUpdated(old, new *clusterv1beta1.ClusterInfoStatus) bool {

	switch new.DistributionInfo.Type {
	case clusterv1beta1.DistributionTypeOCP:
		// sort the slices in distributionInfo to make it comparable using DeepEqual if not update.
		if ocpDistributionInfoUpdated(&old.DistributionInfo.OCP, &new.DistributionInfo.OCP) {
			return true
		}
	}
	return !equality.Semantic.DeepEqual(old, new)
}

func ocpDistributionInfoUpdated(old, new *clusterv1beta1.OCPDistributionInfo) bool {
	sort.SliceStable(new.AvailableUpdates, func(i, j int) bool { return new.AvailableUpdates[i] < new.AvailableUpdates[j] })
	sort.SliceStable(new.VersionAvailableUpdates, func(i, j int) bool {
		return new.VersionAvailableUpdates[i].Version < new.VersionAvailableUpdates[j].Version
	})
	sort.SliceStable(new.VersionHistory, func(i, j int) bool { return new.VersionHistory[i].Version < new.VersionHistory[j].Version })
	return !equality.Semantic.DeepEqual(old, new)
}

func (r *ClusterInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	claimSource := clusterclaim.NewClusterClaimSource(r.ClaimInformer)
	nodeSource := &nodeSource{nodeInformer: r.NodeInformer.Informer()}
	return ctrl.NewControllerManagedBy(mgr).
		Watches(claimSource, clusterclaim.NewClusterClaimEventHandler()).
		Watches(nodeSource, &nodeEventHandler{}).
		For(&clusterv1beta1.ManagedClusterInfo{}).
		Complete(r)
}

// nodeSource is the event source of nodes on managed cluster.
type nodeSource struct {
	nodeInformer cache.SharedIndexInformer
}

var _ source.SyncingSource = &nodeSource{}

func (s *nodeSource) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate) error {
	// all predicates are ignored
	s.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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

func (s *nodeSource) WaitForSync(ctx context.Context) error {
	if ok := cache.WaitForCacheSync(ctx.Done(), s.nodeInformer.HasSynced); !ok {
		return fmt.Errorf("never achieved initial sync")
	}
	return nil
}

// nodeEventHandler maps any event to an empty request
type nodeEventHandler struct{}

var _ handler.EventHandler = &nodeEventHandler{}

func (e *nodeEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}
