package app

import (
	"fmt"
	"net/http"
	"time"

	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"

	"github.com/stolostron/multicloud-operators-foundation/cmd/webhook/app/options"
	"github.com/stolostron/multicloud-operators-foundation/pkg/webhook/mutating"
	"github.com/stolostron/multicloud-operators-foundation/pkg/webhook/validating"
)

func Run(opts *options.Options, stopCh <-chan struct{}) error {
	klog.Info("starting foundation webhook server")

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfigFile)
	if err != nil {
		klog.Errorf("Error building kube config: %s", err.Error())
		return err
	}
	kubeConfig.QPS = opts.QPS
	kubeConfig.Burst = opts.Burst

	kubeClientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		klog.Errorf("Error building kubernetes clientset: %s", err.Error())
		return err
	}

	hiveClient, err := hiveclient.NewForConfig(kubeConfig)
	if err != nil {
		klog.Errorf("Error building hive client: %s", err.Error())
		return err
	}

	clusterClient, err := clusterv1client.NewForConfig(kubeConfig)
	if err != nil {
		klog.Errorf("Error building cluster client: %s", err.Error())
		return err
	}

	kubInformerFactory := informers.NewSharedInformerFactory(kubeClientSet, 10*time.Minute)
	rbInformer := kubInformerFactory.Rbac().V1().RoleBindings()

	clusterInformerFactory := clusterv1informers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
	clusterInformer := clusterInformerFactory.Cluster().V1().ManagedClusters()

	mutatingAh := &mutating.AdmissionHandler{
		Lister:                rbInformer.Lister(),
		SkipOverwriteUserList: opts.SkipOverwriteUserList,
	}
	clusterInformer.Lister()
	validatingAh := &validating.AdmissionHandler{
		KubeClient:    kubeClientSet,
		HiveClient:    hiveClient,
		ClusterLister: clusterInformer.Lister(),
	}

	go kubInformerFactory.Start(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, rbInformer.Informer().HasSynced); !ok {
		klog.Errorf("failed to wait for kubernetes caches to sync")
		return fmt.Errorf("failed to wait for kubernetes caches to sync")
	}

	go clusterInformerFactory.Start(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, clusterInformer.Informer().HasSynced); !ok {
		klog.Errorf("failed to wait for cluster informer caches to sync")
		return fmt.Errorf("failed to wait for cluster informer  caches to sync")
	}

	http.HandleFunc("/mutating", mutatingAh.ServeMutateResource)

	http.HandleFunc("/validating", validatingAh.ServeValidateResource)

	server := &http.Server{
		Addr:      ":8000",
		TLSConfig: options.ConfigTLS(opts),
	}
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		klog.Errorf("Listen server tls error: %+v", err)
		return err
	}

	return nil
}
