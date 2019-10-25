// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package workset

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"k8s.io/klog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	clusterinformers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	clusterlisters "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_listers_generated/clusterregistry/v1alpha1"
	authzutils "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/authz"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	informers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	listers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/listers_generated/mcm/v1alpha1"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils"
	equals "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/equals"
)

// WorkSetController manages the lifecycle of workset
type WorkSetController struct {
	// hcmclientset is a clientset for our own API group
	hcmclientset  clientset.Interface
	kubeclientset kubernetes.Interface

	clusterLister clusterlisters.ClusterLister
	worksetLister listers.WorkSetLister
	workLister    listers.WorkLister

	clusterSynced cache.InformerSynced
	worksetSynced cache.InformerSynced
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

// NewWorkSetController returns a WorkSetController
func NewWorkSetController(
	hcmclientset clientset.Interface,
	kubeclientset kubernetes.Interface,
	informerFactory informers.SharedInformerFactory,
	clusterInformerFactory clusterinformers.SharedInformerFactory,
	enableRBAC bool,
	stopCh <-chan struct{}) *WorkSetController {

	clusterInformer := clusterInformerFactory.Clusterregistry().V1alpha1().Clusters()
	worksetInformer := informerFactory.Mcm().V1alpha1().WorkSets()
	workInformer := informerFactory.Mcm().V1alpha1().Works()

	controller := &WorkSetController{
		hcmclientset:  hcmclientset,
		kubeclientset: kubeclientset,
		clusterLister: clusterInformer.Lister(),
		worksetLister: worksetInformer.Lister(),
		workLister:    workInformer.Lister(),
		clusterSynced: clusterInformer.Informer().HasSynced,
		worksetSynced: worksetInformer.Informer().HasSynced,
		workSynced:    workInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "worksetController"),
		stopCh:        stopCh,
		enableRBAC:    enableRBAC,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	worksetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueWorkSet,
		UpdateFunc: func(old, new interface{}) {
			oldworkset := old.(*v1alpha1.WorkSet)
			newworkset := new.(*v1alpha1.WorkSet)
			if controller.needUpdate(oldworkset, newworkset) {
				controller.enqueueWorkSet(new)
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
func (w *WorkSetController) Run() error {
	defer utilruntime.HandleCrash()
	defer w.workqueue.ShutDown()

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for hcm informer caches to sync")
	if ok := cache.WaitForCacheSync(w.stopCh, w.clusterSynced, w.worksetSynced, w.workSynced); !ok {
		return fmt.Errorf("failed to wait for hcm caches to sync")
	}

	go wait.Until(w.runWorker, time.Second, w.stopCh)

	<-w.stopCh
	klog.Info("Shutting controller")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (w *WorkSetController) runWorker() {
	for w.processNextWorkItem() {
	}
	return
}

func (w *WorkSetController) processNextWorkItem() bool {
	obj, shutdown := w.workqueue.Get()

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
		defer w.workqueue.Done(obj)
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
			w.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced
		if err := w.processWorkSet(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		w.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// processWork process work and update work result
func (w *WorkSetController) processWorkSet(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	workset, err := w.worksetLister.WorkSets(namespace).Get(name)
	if err != nil {
		// The Work resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("workset '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	clusterSelector, err := utils.ConvertLabels(workset.Spec.ClusterSelector)
	if err != nil {
		return fmt.Errorf("workset convert labels err %v", err)
	}

	clusters, err := w.clusterLister.List(clusterSelector)
	if err != nil {
		return fmt.Errorf("workset list cluster err %v", err)
	}

	filteredClusters := utils.FilterHealthyClusters(clusters)
	if w.enableRBAC {
		filteredClusters = authzutils.FilterClusterByUserIdentity(workset, filteredClusters, w.kubeclientset, "works", "create")
	}

	clusterToWorks, err := w.getClustersToWorks(workset)
	if err != nil {
		return fmt.Errorf("workset convert labels err %v", err)
	}

	clustersNeedingWorks, worksToDelete, worksToUpdate := w.worksShouldBeOnClusters(workset, clusterToWorks, filteredClusters)

	// Create work items on each cluster
	utils.BatchHandle(len(clustersNeedingWorks), func(i int) error {
		cluster := clustersNeedingWorks[i]
		workName := workset.Name + "-" + cluster.Name
		owners := utils.AddOwnersLabel("", "clusters", cluster.Name, cluster.Namespace)
		owners = utils.AddOwnersLabel(owners, "worksets", workset.Name, workset.Namespace)
		template := workset.Spec.Template.DeepCopy()
		work := &v1alpha1.Work{
			ObjectMeta: template.ObjectMeta,
			Spec:       template.Spec,
		}

		if work.ObjectMeta.Labels == nil {
			work.ObjectMeta.Labels = map[string]string{}
		}
		if work.ObjectMeta.Annotations == nil {
			work.ObjectMeta.Annotations = map[string]string{}
		}

		work.ObjectMeta.GenerateName = workName
		work.ObjectMeta.Namespace = cluster.Namespace
		work.ObjectMeta.Labels[mcm.WorkSetLabel] = fmt.Sprintf("%s.%s", workset.Namespace, workset.Name)
		work.ObjectMeta.Annotations[mcm.OwnersLabel] = owners
		work.Spec.Cluster = corev1.LocalObjectReference{
			Name: cluster.Name,
		}
		_, e := w.hcmclientset.McmV1alpha1().Works(cluster.Namespace).Create(work)
		if e != nil {
			klog.Errorf("Failed to create work %s on cluster %s: %v", work.Name, cluster.Name, e)
		}
		return nil
	})

	utils.BatchHandle(len(worksToUpdate), func(i int) error {
		workToUpdate := worksToUpdate[i]
		_, err := w.hcmclientset.McmV1alpha1().Works(workToUpdate.Namespace).Update(workToUpdate)
		if err != nil {
			klog.Errorf("Failed to update work %s: %v", workToUpdate.Name, err)
		}
		return nil
	})

	utils.BatchHandle(len(worksToDelete), func(i int) error {
		workToDelete := worksToDelete[i]
		err := w.hcmclientset.McmV1alpha1().Works(workToDelete.Namespace).Delete(workToDelete.Name, &metav1.DeleteOptions{})
		if err != nil {
			klog.Errorf("Failed to delete work %s", workToDelete.Name)
		}
		return nil
	})

	return w.updateWorkSetStatus(workset, filteredClusters, clusterToWorks)
}

func (w *WorkSetController) worksShouldBeOnClusters(
	workset *v1alpha1.WorkSet,
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
			if updatedwork, update := w.updateWorkByWorkset(workset, work); update {
				workToUpdate = append(workToUpdate, updatedwork)
			}
		}
	}

	return clustersNeedingWorks, worksToDelete, workToUpdate
}

func (w *WorkSetController) getClustersToWorks(workset *v1alpha1.WorkSet) (map[string][]*v1alpha1.Work, error) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			mcm.WorkSetLabel: workset.Namespace + "." + workset.Name,
		},
	}

	workSelector, err := utils.ConvertLabels(selector)
	if err != nil {
		return nil, err
	}

	works, err := w.workLister.List(workSelector)
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

func (w *WorkSetController) updateWorkByWorkset(workset *v1alpha1.WorkSet, work *v1alpha1.Work) (*v1alpha1.Work, bool) {
	update := false
	updateWork := work.DeepCopy()

	if w.needUpdateWork(workset, updateWork) {
		clusterRef := updateWork.Spec.Cluster
		updateWork.Spec = workset.Spec.Template.Spec
		updateWork.Spec.Cluster = clusterRef
		update = true
	}

	return updateWork, update
}

func (w *WorkSetController) updateWorkSetStatus(
	oldworkset *v1alpha1.WorkSet,
	clusters []*clusterv1alpha1.Cluster,
	clusterToWorks map[string][]*v1alpha1.Work,
) error {
	// Deepcopy workset
	workset := oldworkset.DeepCopy()
	status := oldworkset.Status.DeepCopy()
	if status.Status == v1alpha1.WorkCompleted {
		return nil
	}

	finishedWorkNum := 0
	reason := ""
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
		if work.Status.Type == v1alpha1.WorkFailed {
			reason = fmt.Sprintf("%s%s(%s); ", reason, work.Status.Reason, cluster.Name)
		}
		finishedWorkNum = finishedWorkNum + 1
	}

	if len(clusters) <= finishedWorkNum {
		status.Status = v1alpha1.WorkCompleted
		status.Reason = reason
	}

	err := w.retryUpdateWorkSetStatus(workset, status)
	if err != nil {
		return err
	}

	return nil
}

func (w *WorkSetController) retryUpdateWorkSetStatus(
	workset *v1alpha1.WorkSet, status *v1alpha1.WorkSetStatus) error {
	// don't wait due to limited number of clients, but backoff after the default number of steps
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		workset.Status = *status
		_, updateErr := w.hcmclientset.McmV1alpha1().WorkSets(workset.Namespace).UpdateStatus(workset)
		if updateErr == nil {
			return nil
		}
		if updated, err := w.worksetLister.WorkSets(workset.Namespace).Get(workset.Name); err == nil {
			// make a copy so we don't mutate the shared cache
			workset = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated workset %s/%s from lister: %v", workset.Namespace, workset.Name, err))
		}

		return updateErr
	})
}

func (w *WorkSetController) addWork(obj interface{}) {
	work := obj.(*v1alpha1.Work)
	if work.Status.Type != "" {
		w.enqueueWorkSetFromWork(work)
	}
}

func (w *WorkSetController) updateWork(old, new interface{}) {
	newWork := new.(*v1alpha1.Work)
	oldWork := old.(*v1alpha1.Work)

	// enqueu work if it is transfer from pending to completed or failed
	if (newWork.Status.Type == v1alpha1.WorkCompleted || newWork.Status.Type == v1alpha1.WorkFailed) && oldWork.Status.Type == "" {
		w.enqueueWorkSetFromWork(newWork)
	}
}

func (w *WorkSetController) deleteWork(old interface{}) {
	oldWork := old.(*v1alpha1.Work)
	if oldWork.Status.Type != v1alpha1.WorkCompleted {
		w.enqueueWorkSetFromWork(oldWork)
	}
}

func (w *WorkSetController) updateCluster(old, new interface{}) {
	oldCluster := old.(*clusterv1alpha1.Cluster)
	newCluster := new.(*clusterv1alpha1.Cluster)

	if len(oldCluster.Status.Conditions) > 0 && oldCluster.Status.Conditions[0].Type == newCluster.Status.Conditions[0].Type {
		return
	}

	worksets, _ := w.worksetLister.List(labels.Everything())
	for _, workset := range worksets {
		if !utils.MatchLabelForLabelSelector(oldCluster.Labels, workset.Spec.ClusterSelector) {
			continue
		}

		if workset.Status.Status != v1alpha1.WorkCompleted {
			w.enqueueWorkSet(workset)
		}
	}
}

// enqueueWork takes a Work resource and converts it into a name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Work.
func (w *WorkSetController) enqueueWorkSet(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	w.workqueue.AddRateLimited(key)
}

func (w *WorkSetController) enqueueWorkSetFromWork(work *v1alpha1.Work) {
	key, ok := work.Labels[mcm.WorkSetLabel]
	if !ok {
		return
	}

	worksetNamespacedName := strings.Split(key, ".")
	if len(worksetNamespacedName) < 2 {
		return
	}

	key = strings.Join(worksetNamespacedName, "/")
	w.workqueue.Add(key)

	return
}

func (w *WorkSetController) needUpdateWork(workset *v1alpha1.WorkSet, work *v1alpha1.Work) bool {
	return !equals.EqualWorkSpec(&workset.Spec.Template.Spec, &work.Spec)
}

func (w *WorkSetController) needUpdate(newworkset, oldworkset *v1alpha1.WorkSet) bool {
	if !reflect.DeepEqual(newworkset.Spec.ClusterSelector, oldworkset.Spec.ClusterSelector) {
		return true
	}

	if !equals.EqualWorkSpec(&newworkset.Spec.Template.Spec, &oldworkset.Spec.Template.Spec) {
		return true
	}

	return false
}
