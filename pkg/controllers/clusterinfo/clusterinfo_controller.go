package clusterinfo

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"

	clusterinfov1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterFinalizerName = "managedclusterinfo.finalizers.open-cluster-management.io"
)

var (
	logCertSecretNamespace string
	logCertSecretName      string
)

type ClusterInfoReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func SetupWithManager(mgr manager.Manager, logCertSecret string) error {
	var err error

	if len(logCertSecret) != 0 {
		logCertSecretNamespace, logCertSecretName, err = cache.SplitMetaNamespaceKey(logCertSecret)
		if err != nil {
			return err
		}
		if logCertSecretNamespace == "" {
			logCertSecretNamespace, err = utils.GetComponentNamespace()
			if err != nil {
				return err
			}
		}
	} else {
		klog.Warning("The log cert secret is not specified, ignore it")
	}

	if err = add("clusterinfo-controller", mgr, newClusterInfoReconciler(mgr)); err != nil {
		return err
	}
	if err = add("clusterdetector-controller", mgr, newAutoDetectReconciler(mgr)); err != nil {
		return err
	}
	if err = add("clustercapcity-controller", mgr, newCapacityReconciler(mgr)); err != nil {
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newClusterInfoReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ClusterInfoReconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(name string, mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(name, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1.ManagedCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &clusterinfov1beta1.ManagedClusterInfo{}},
		&handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(
		handler.MapFunc(func(a client.Object) []reconcile.Request {
			certSecret, ok := a.(*corev1.Secret)
			if !ok {
				// not a secret, returning empty
				klog.Error("clusterinfo handler received non-secret object")
				return []reconcile.Request{}
			}

			if certSecret.Name != logCertSecretName || certSecret.Namespace != logCertSecretNamespace {
				return []reconcile.Request{}
			}

			managedClusterInfoList := &clusterinfov1beta1.ManagedClusterInfoList{}
			err := mgr.GetClient().List(context.TODO(), managedClusterInfoList)
			if err != nil {
				klog.Error("Could not list managedClusterInfo", err)
			}
			var requests []reconcile.Request
			for _, managedClusterInfo := range managedClusterInfoList.Items {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      managedClusterInfo.Name,
						Namespace: managedClusterInfo.Namespace,
					},
				})
			}
			return requests
		}),
	))

	return err
}

func (r *ClusterInfoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &clusterv1.ManagedCluster{}

	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if !cluster.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
			err := r.deleteClusterInfo(cluster.Name)
			if err != nil {
				return reconcile.Result{}, err
			}

			patch := client.MergeFrom(cluster.DeepCopy())
			cluster.Finalizers = utils.RemoveString(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
			return reconcile.Result{}, r.client.Patch(ctx, cluster, patch)
		}

		return reconcile.Result{}, nil
	}

	if !utils.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
		patch := client.MergeFrom(cluster.DeepCopy())
		cluster.Finalizers = append(cluster.Finalizers, clusterFinalizerName)
		return reconcile.Result{}, r.client.Patch(ctx, cluster, patch)
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Name}, clusterInfo)
	switch {
	case errors.IsNotFound(err):
		return ctrl.Result{}, r.client.Create(ctx, r.newClusterInfoByManagedCluster(cluster))
	case err != nil:
		return ctrl.Result{}, err
	}

	update := false
	endpoint := ""
	if len(cluster.Spec.ManagedClusterClientConfigs) != 0 {
		endpoint = cluster.Spec.ManagedClusterClientConfigs[0].URL
		if endpoint != "" && endpoint != clusterInfo.Spec.MasterEndpoint {
			clusterInfo.Spec.MasterEndpoint = endpoint
			update = true
		}
	}

	caData, err := r.getLogCA()
	if err != nil {
		klog.Errorf("failed to get log CA. %v", err)
	}

	if caData != nil && !reflect.DeepEqual(caData, clusterInfo.Spec.LoggingCA) {
		clusterInfo.Spec.LoggingCA = caData
		update = true
	}

	if update {
		return ctrl.Result{}, r.client.Update(ctx, clusterInfo)
	}

	// TODO: the conditions of managed cluster need to be deprecated.
	clusterConditions := cluster.Status.Conditions
	newClusterInfo := clusterInfo.DeepCopy()

	// empty node list in clusterInfo status when cluster is offline
	if utils.ClusterIsOffLine(clusterConditions) {
		newClusterInfo.Status.NodeList = []clusterinfov1beta1.NodeStatus{}
	}
	newClusterInfo.Status.Conditions = clusterConditions
	syncedCondition := meta.FindStatusCondition(clusterInfo.Status.Conditions, clusterinfov1beta1.ManagedClusterInfoSynced)
	if syncedCondition != nil {
		newClusterInfo.Status.Conditions = append(newClusterInfo.Status.Conditions, *syncedCondition)
	}

	if !reflect.DeepEqual(newClusterInfo.Status, clusterInfo.Status) {
		err = r.client.Status().Update(ctx, newClusterInfo)
		if err != nil {
			klog.Warningf("will reconcile since failed to update ManagedClusterInfo status %v, %v", cluster.Name, err)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *ClusterInfoReconciler) deleteClusterInfo(name string) error {
	err := r.client.Delete(context.Background(), &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
	})
	return client.IgnoreNotFound(err)
}

func (r *ClusterInfoReconciler) newClusterInfoByManagedCluster(cluster *clusterv1.ManagedCluster) *clusterinfov1beta1.ManagedClusterInfo {
	endpoint := ""
	if len(cluster.Spec.ManagedClusterClientConfigs) != 0 {
		endpoint = cluster.Spec.ManagedClusterClientConfigs[0].URL
	}

	caData, err := r.getLogCA()
	if err != nil {
		klog.Errorf("failed to get log CA. %v", err)
	}

	return &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Name,
			Labels:    cluster.Labels,
		},
		Spec: clusterinfov1beta1.ClusterInfoSpec{
			MasterEndpoint: endpoint,
			LoggingCA:      caData,
		},
		Status: clusterinfov1beta1.ClusterInfoStatus{
			Conditions: cluster.Status.Conditions,
		},
	}
}

func (r *ClusterInfoReconciler) getLogCA() ([]byte, error) {
	if len(logCertSecretName) == 0 {
		// the log cert secret is not specified, ignore it
		return nil, nil
	}

	logCertSecret := &corev1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: logCertSecretName, Namespace: logCertSecretNamespace}, logCertSecret)
	if err != nil {
		klog.Errorf("failed to get log cert secret %v/%v: %v", logCertSecretNamespace, logCertSecretName, err)
		return nil, err
	}
	caData := logCertSecret.Data["ca.crt"]
	return caData, nil
}
