package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stolostron/cluster-lifecycle-api/helpers/tlsprofile"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stolostron/multicloud-operators-foundation/cmd/webhook/app/options"
	"github.com/stolostron/multicloud-operators-foundation/pkg/webhook/mutating"
	"github.com/stolostron/multicloud-operators-foundation/pkg/webhook/validating"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(hivev1.AddToScheme(scheme))
}

func Run(opts *options.Options, externalStopCh <-chan struct{}) error {
	klog.Info("starting foundation webhook server")

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create internal stop channel that we can control for graceful shutdown
	// This allows us to properly stop all components when TLS profile changes
	stopCh := make(chan struct{})

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfigFile)
	if err != nil {
		klog.Errorf("Error building kube config: %s", err.Error())
		return err
	}
	kubeConfig.QPS = opts.QPS
	kubeConfig.Burst = opts.Burst

	// Start TLS profile watcher to detect changes and trigger graceful restart
	// Returns nil on non-OpenShift clusters, error only for real problems
	if err := tlsprofile.StartTLSProfileWatcher(ctx, kubeConfig, cancel); err != nil {
		klog.Errorf("Failed to start TLS profile watcher: %v", err)
		return err
	}

	kubeClientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		klog.Errorf("Error building kubernetes clientset: %s", err.Error())
		return err
	}

	hiveClient, err := client.New(kubeConfig, client.Options{Scheme: scheme})
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

	// Get TLS configuration from OpenShift APIServer CR
	// Returns default TLS 1.2 on non-OpenShift, error only for real problems
	tlsConfig, err := options.ConfigTLS(opts, kubeConfig)
	if err != nil {
		klog.Errorf("Failed to configure TLS: %v", err)
		return err
	}

	server := &http.Server{
		Addr:      ":8000",
		TLSConfig: tlsConfig,
	}

	// Merge shutdown signals and trigger server shutdown
	go func() {
		select {
		case <-externalStopCh:
			klog.Info("Received stop signal, shutting down webhook server...")
		case <-ctx.Done():
			klog.Info("TLS profile changed, shutting down webhook server for restart...")
		}

		// Stop informers
		close(stopCh)

		// Graceful shutdown of HTTP server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			klog.Errorf("Error during server shutdown: %v", err)
		}
	}()

	// Run server in main goroutine - blocks until Shutdown() is called
	klog.Info("Starting webhook server on :8000")
	if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("webhook server failed: %w", err)
	}

	klog.Info("Webhook server shutdown complete")
	return nil
}
