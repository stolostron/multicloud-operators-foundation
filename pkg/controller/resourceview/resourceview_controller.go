// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package resourceview

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/klog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	v1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	clusterinformers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	clusterlisters "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_listers_generated/clusterregistry/v1alpha1"
	authzutils "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/authz"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"

	clientset "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	informers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	listers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/listers_generated/mcm/v1alpha1"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils"
	equals "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/equals"
)

// Controller manages the lifecycle of ResourceView
type Controller struct {
	// hcmclientset is a clientset for our own API group
	hcmclientset clientset.Interface

	kubeclientset kubernetes.Interface

	clusterLister clusterlisters.ClusterLister
	viewLister    listers.ResourceViewLister
	workLister    listers.WorkLister

	clusterSynced cache.InformerSynced
	viewSynced    cache.InformerSynced
	workSynced    cache.InformerSynced

	enableRBAC bool
	stopCh     <-chan struct{}

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
}

// NewController returns a resource view controller
func NewController(
	hcmclientset clientset.Interface,
	kubeclientset kubernetes.Interface,
	informerFactory informers.SharedInformerFactory,
	clusterInformerFactory clusterinformers.SharedInformerFactory,
	enableRBAC bool,
	stopCh <-chan struct{}) *Controller {
	clusterInformer := clusterInformerFactory.Clusterregistry().V1alpha1().Clusters()
	viewInformer := informerFactory.Mcm().V1alpha1().ResourceViews()
	workInformer := informerFactory.Mcm().V1alpha1().Works()

	controller := &Controller{
		hcmclientset:  hcmclientset,
		kubeclientset: kubeclientset,
		clusterLister: clusterInformer.Lister(),
		viewLister:    viewInformer.Lister(),
		workLister:    workInformer.Lister(),
		clusterSynced: clusterInformer.Informer().HasSynced,
		viewSynced:    viewInformer.Informer().HasSynced,
		workSynced:    workInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "resourceviewController"),
		enableRBAC:    enableRBAC,
		stopCh:        stopCh,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	viewInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(new interface{}) {
			controller.enqueueView(new)
		},
		UpdateFunc: func(old, new interface{}) {
			oldview := old.(*v1alpha1.ResourceView)
			newview := new.(*v1alpha1.ResourceView)
			if controller.needsUpdate(oldview, newview) {
				controller.enqueueView(new)
			}
		},
	})

	workInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addWork,
		UpdateFunc: controller.updateWork,
		DeleteFunc: controller.deleteWork,
	})

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: controller.updateCluster,
	})
	return controller
}

// Run is the main run loop of kluster server
func (rv *Controller) Run() {
	defer utilruntime.HandleCrash()
	defer rv.workqueue.ShutDown()

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for hcm informer caches to sync")
	if ok := cache.WaitForCacheSync(rv.stopCh, rv.clusterSynced, rv.viewSynced, rv.workSynced); !ok {
		klog.Errorf("failed to wait for hcm caches to sync")
		return
	}

	go wait.Until(rv.runWorker, time.Second, rv.stopCh)

	<-rv.stopCh
	klog.Info("Shutting controller")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (rv *Controller) runWorker() {
	for rv.processNextWorkItem() {
	}
}

func (rv *Controller) processNextWorkItem() bool {
	obj, shutdown := rv.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer rv.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			rv.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := rv.processView(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		rv.workqueue.Forget(obj)
		klog.V(5).Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// processWork process work and update work result
func (rv *Controller) processView(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	view, err := rv.viewLister.ResourceViews(namespace).Get(name)
	if err != nil {
		// The Work resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("view '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// To reduce overhead on controller, do not update immediately, wait until updateInterval comes.
	if view.Spec.Mode == v1alpha1.PeriodicResourceUpdate {
		viewCondition := getViewCondition(view)
		current := metav1.Now()
		interval := view.Spec.UpdateIntervalSeconds
		if len(view.Status.Results) > 0 &&
			!current.After(viewCondition.LastUpdateTime.Add(time.Duration(interval/2)*time.Second)) {
			return nil
		}
	}

	clusterSelector, err := utils.ConvertLabels(view.Spec.ClusterSelector)
	if err != nil {
		return err
	}

	clusters, err := rv.clusterLister.List(clusterSelector)
	if err != nil {
		return err
	}

	filteredClusters := utils.FilterHealthyClusters(clusters)
	if rv.enableRBAC {
		filteredClusters = authzutils.FilterClusterByUserIdentity(view, filteredClusters, rv.kubeclientset, "works", "create")
	}

	clusterToWorks, err := rv.getClustersToWorks(view)
	if err != nil {
		return err
	}

	clustersNeedingWorks, worksToDelete, worksToUpdate := rv.worksShouldBeOnClusters(view, clusterToWorks, filteredClusters)
	// Create work items on each cluster
	utils.BatchHandle(len(clustersNeedingWorks), func(i int) {
		cluster := clustersNeedingWorks[i]
		workName := "view-" + view.Name
		owners := utils.AddOwnersLabel("", "clusters", cluster.Name, cluster.Namespace)
		owners = utils.AddOwnersLabel(owners, "resourceviews", view.Name, view.Namespace)
		work := &v1alpha1.Work{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: workName,
				Namespace:    cluster.Namespace,
				Labels: map[string]string{
					mcm.ViewLabel: fmt.Sprintf("%s.%s", view.Namespace, view.Name),
				},
				Annotations: map[string]string{
					mcm.OwnersLabel: owners,
				},
			},
			Spec: v1alpha1.WorkSpec{
				Type: v1alpha1.ResourceWorkType,
				Cluster: corev1.LocalObjectReference{
					Name: cluster.Name,
				},
				Scope: v1alpha1.ResourceFilter{
					APIGroup:              view.Spec.Scope.APIGroup,
					LabelSelector:         view.Spec.Scope.LabelSelector.DeepCopy(),
					FieldSelector:         view.Spec.Scope.FieldSelector,
					ResourceType:          view.Spec.Scope.Resource,
					Name:                  view.Spec.Scope.ResourceName,
					NameSpace:             view.Spec.Scope.NameSpace,
					Mode:                  view.Spec.Mode,
					UpdateIntervalSeconds: view.Spec.UpdateIntervalSeconds,
					ServerPrint:           view.Spec.SummaryOnly,
				},
			},
		}

		_, e := rv.hcmclientset.McmV1alpha1().Works(cluster.Namespace).Create(work)
		if e != nil {
			klog.Errorf("Failed to create work %s on cluster %s, error: %+v", work.Name, cluster.Name, e)
		}
	})

	utils.BatchHandle(len(worksToUpdate), func(i int) {
		workToUpdate := worksToUpdate[i]
		_, err := rv.hcmclientset.McmV1alpha1().Works(workToUpdate.Namespace).Update(workToUpdate)
		if err != nil {
			klog.Errorf("Failed to update work %s error: %v", workToUpdate.Name, err)
		}
	})

	utils.BatchHandle(len(worksToDelete), func(i int) {
		workToDelete := worksToDelete[i]
		err := rv.hcmclientset.McmV1alpha1().Works(workToDelete.Namespace).Delete(workToDelete.Name, &metav1.DeleteOptions{})
		if err != nil {
			klog.Errorf("Failed to delete work %s, error: %v", workToDelete.Name, err)
		}
	})

	return rv.updateViewStatus(view, filteredClusters, clusterToWorks)
}

func (rv *Controller) worksShouldBeOnClusters(
	view *v1alpha1.ResourceView,
	clusterToWorks map[string][]*v1alpha1.Work,
	clusters []*clusterv1alpha1.Cluster,
) (clustersNeedingWorks []*clusterv1alpha1.Cluster, worksToDelete, workToUpdate []*v1alpha1.Work) {
	for _, cluster := range clusters {
		if _, ok := clusterToWorks[cluster.Namespace+"/"+cluster.Name]; !ok {
			clustersNeedingWorks = append(clustersNeedingWorks, cluster)
		}
	}

	for key, works := range clusterToWorks {
		deleteWorks := true
		for _, cluster := range clusters {
			if key == cluster.Namespace+"/"+cluster.Name {
				deleteWorks = false
				break
			}
		}
		if deleteWorks {
			worksToDelete = append(worksToDelete, works...)
			continue
		}

		for _, work := range works {
			if updatedwork, update := rv.updateWorkByView(view, work); update {
				workToUpdate = append(workToUpdate, updatedwork)
			}
		}
	}

	return clustersNeedingWorks, worksToDelete, workToUpdate
}

func (rv *Controller) getClustersToWorks(view *v1alpha1.ResourceView) (map[string][]*v1alpha1.Work, error) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			mcm.ViewLabel: view.Namespace + "." + view.Name,
		},
	}

	workSelector, err := utils.ConvertLabels(selector)
	if err != nil {
		return nil, err
	}
	works, err := rv.workLister.List(workSelector)
	if err != nil {
		return nil, err
	}

	clusterToWorks := make(map[string][]*v1alpha1.Work)
	for _, work := range works {
		cluster := work.Spec.Cluster.Name
		clusterToWorks[work.Namespace+"/"+cluster] = append(clusterToWorks[cluster], work)
	}

	return clusterToWorks, nil
}

func (rv *Controller) updateViewStatus(
	oldview *v1alpha1.ResourceView,
	clusters []*clusterv1alpha1.Cluster,
	clusterToWorks map[string][]*v1alpha1.Work,
) error {
	view := oldview.DeepCopy()
	status := oldview.Status.DeepCopy()

	// skip update if view is done.
	if getViewCondition(view).Type == v1alpha1.WorkCompleted {
		return nil
	}

	finishedWorkNum := 0
	for _, cluster := range clusters {
		works, ok := clusterToWorks[cluster.Namespace+"/"+cluster.Name]
		if !ok || len(works) != 1 {
			klog.V(5).Infof("cannot find matched works")
			continue
		}

		work := works[0]
		if work.Status.Type == "" {
			continue
		}
		finishedWorkNum++
	}

	if view.Spec.Mode == v1alpha1.PeriodicResourceUpdate {
		status.Conditions = createViewContidion(view, v1alpha1.WorkProcessing)
	} else if len(clusters) <= finishedWorkNum {
		status.Conditions = createViewContidion(view, v1alpha1.WorkCompleted)
	}

	if len(status.Conditions) == 0 {
		return nil
	}

	err := rv.retryUpdateViewStatus(view, status)
	if err != nil {
		return err
	}

	return nil
}

func (rv *Controller) retryUpdateViewStatus(
	view *v1alpha1.ResourceView, status *v1alpha1.ResourceViewStatus) error {
	// don't wait due to limited number of clients, but backoff after the default number of steps
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		view.Status = *status
		_, updateErr := rv.hcmclientset.McmV1alpha1().ResourceViews(view.Namespace).UpdateStatus(view)
		if updateErr == nil {
			return nil
		}

		fieldSelector := fields.OneTermEqualSelector("metadata.name", view.Name).String()
		viewlist, err := rv.hcmclientset.McmV1alpha1().ResourceViews(view.Namespace).List(metav1.ListOptions{FieldSelector: fieldSelector})
		if err != nil || len(viewlist.Items) != 1 {
			utilruntime.HandleError(fmt.Errorf("error getting updated resourceview %s/%s from lister: %v", view.Namespace, view.Name, err))
		} else {
			view = (&viewlist.Items[0]).DeepCopy()
		}

		return updateErr
	})
}

// enqueueWork takes a Work resource and converts it into a name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Work.
func (rv *Controller) enqueueView(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	rv.workqueue.Add(key)
}

func (rv *Controller) addWork(obj interface{}) {
	work := obj.(*v1alpha1.Work)

	if work.Spec.Type != v1alpha1.ResourceWorkType {
		return
	}

	if work.Status.Type != "" {
		rv.enqueueViewFromWork(work)
	}
}

func (rv *Controller) updateWork(oldObj, newObj interface{}) {
	newWork := newObj.(*v1alpha1.Work)
	oldWork := oldObj.(*v1alpha1.Work)

	if newWork.Spec.Type != v1alpha1.ResourceWorkType {
		return
	}

	// enqueu work if it is in processing state
	if newWork.Status.Type == v1alpha1.WorkProcessing {
		rv.enqueueViewFromWork(newWork)
	}

	// enqueu work if it is transfer from pending to completed or failed
	if (newWork.Status.Type == v1alpha1.WorkCompleted || newWork.Status.Type == v1alpha1.WorkFailed) && oldWork.Status.Type == "" {
		rv.enqueueViewFromWork(newWork)
	}
}

func (rv *Controller) deleteWork(old interface{}) {
	oldWork := old.(*v1alpha1.Work)
	if oldWork.Spec.Type != v1alpha1.ResourceWorkType {
		return
	}

	if oldWork.Status.Type != v1alpha1.WorkCompleted {
		rv.enqueueViewFromWork(oldWork)
	}
}

func (rv *Controller) updateCluster(oldObj, newObj interface{}) {
	oldCluster := oldObj.(*clusterv1alpha1.Cluster)
	newCluster := newObj.(*clusterv1alpha1.Cluster)

	if len(oldCluster.Status.Conditions) > 0 &&
		oldCluster.Status.Conditions[0].Type == newCluster.Status.Conditions[0].Type {
		return
	}

	views, _ := rv.viewLister.List(labels.Everything())
	for _, view := range views {
		if !utils.MatchLabelForLabelSelector(oldCluster.Labels, view.Spec.ClusterSelector) {
			continue
		}

		if getViewCondition(view).Type != v1alpha1.WorkCompleted {
			rv.enqueueView(view)
		}
	}
}

func (rv *Controller) updateWorkByView(view *v1alpha1.ResourceView, work *v1alpha1.Work) (*v1alpha1.Work, bool) {
	update := false
	updateWork := work.DeepCopy()

	if work.Spec.Scope.Mode != view.Spec.Mode {
		updateWork.Spec.Scope.Mode = view.Spec.Mode
		update = true
	}

	if work.Spec.Scope.ServerPrint != view.Spec.SummaryOnly {
		updateWork.Spec.Scope.ServerPrint = view.Spec.SummaryOnly
		update = true
	}

	if work.Spec.Scope.UpdateIntervalSeconds != view.Spec.UpdateIntervalSeconds {
		updateWork.Spec.Scope.UpdateIntervalSeconds = view.Spec.UpdateIntervalSeconds
		update = true
	}

	if work.Spec.Scope.FieldSelector != view.Spec.Scope.FieldSelector {
		updateWork.Spec.Scope.FieldSelector = view.Spec.Scope.FieldSelector
		update = true
	}

	if work.Spec.Scope.LabelSelector.String() != view.Spec.Scope.LabelSelector.String() {
		updateWork.Spec.Scope.LabelSelector = view.Spec.Scope.LabelSelector
		update = true
	}

	if work.Spec.Scope.ResourceType != view.Spec.Scope.Resource {
		updateWork.Spec.Scope.ResourceType = view.Spec.Scope.Resource
		update = true
	}

	if work.Spec.Scope.NameSpace != view.Spec.Scope.NameSpace {
		updateWork.Spec.Scope.NameSpace = view.Spec.Scope.NameSpace
		update = true
	}

	if work.Spec.Scope.Name != view.Spec.Scope.ResourceName {
		updateWork.Spec.Scope.Name = view.Spec.Scope.ResourceName
		update = true
	}

	return updateWork, update
}

func (rv *Controller) enqueueViewFromWork(work *v1alpha1.Work) {
	key, ok := work.Labels[mcm.ViewLabel]
	if !ok {
		return
	}

	viewNamespacedName := strings.Split(key, ".")
	if len(viewNamespacedName) < 2 {
		return
	}

	key = strings.Join(viewNamespacedName, "/")
	rv.workqueue.Add(key)
}

func (rv *Controller) needsUpdate(oldview, newview *v1alpha1.ResourceView) bool {
	// If it has a ControllerRef, that's all that matters.
	if oldview.Spec.Mode != newview.Spec.Mode {
		return true
	}

	if oldview.Spec.UpdateIntervalSeconds != newview.Spec.UpdateIntervalSeconds {
		return true
	}

	if oldview.Spec.SummaryOnly != newview.Spec.SummaryOnly {
		return true
	}

	return !equalResourceViewSpec(oldview.Spec.Scope, newview.Spec.Scope)
}

func equalResourceViewSpec(oldscope, newscope v1alpha1.ViewFilter) bool {
	if oldscope.APIGroup != newscope.APIGroup {
		return false
	}

	if oldscope.FieldSelector != newscope.FieldSelector {
		return false
	}

	if !equals.EqualLabelSelector(oldscope.LabelSelector, newscope.LabelSelector) {
		return false
	}

	if oldscope.NameSpace != newscope.NameSpace {
		return false
	}

	if oldscope.Resource != newscope.Resource {
		return false
	}

	if oldscope.ResourceName != newscope.ResourceName {
		return false
	}

	return true
}

func getViewCondition(view *v1alpha1.ResourceView) v1alpha1.ViewCondition {
	if len(view.Status.Conditions) == 0 {
		return v1alpha1.ViewCondition{}
	}

	return view.Status.Conditions[len(view.Status.Conditions)-1]
}

func createViewContidion(view *v1alpha1.ResourceView, conditionType v1alpha1.WorkStatusType) []v1alpha1.ViewCondition {
	conditions := view.Status.Conditions
	if conditions == nil {
		conditions = []v1alpha1.ViewCondition{}
	}

	condition := v1alpha1.ViewCondition{
		Type:           conditionType,
		LastUpdateTime: metav1.Now(),
	}

	if len(conditions) == 0 || conditions[len(conditions)-1].Type != conditionType {
		conditions = append(conditions, condition)
	} else {
		conditions[len(conditions)-1] = condition
	}

	return conditions
}
