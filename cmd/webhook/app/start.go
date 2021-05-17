package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/webhook/clusterset"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/webhook/useridentity"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/webhook/app/options"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
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

	informerFactory := informers.NewSharedInformerFactory(kubeClientSet, 10*time.Minute)
	informer := informerFactory.Rbac().V1().RoleBindings()

	mutatingAh := &useridentity.AdmissionHandler{
		Lister: informer.Lister(),
	}

	validatingAh := &clusterset.AdmissionHandler{
		KubeClient: kubeClientSet,
	}

	go informerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, informer.Informer().HasSynced); !ok {
		klog.Errorf("failed to wait for kubernetes caches to sync")
		return fmt.Errorf("failed to wait for kubernetes caches to sync")
	}

	http.HandleFunc("/mutating", mutatingAh.ServeMutateResource)

	http.HandleFunc("/validating", validatingAh.ServerValidateResource)

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
