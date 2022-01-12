package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/cmd/webhook/app/options"
	"k8s.io/client-go/dynamic"
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

	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		klog.Errorf("Error building dynamic client: %s", err.Error())
		return err
	}

	kubeClientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		klog.Errorf("Error building kubernetes clientset: %s", err.Error())
		return err
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClientSet, 10*time.Minute)
	informer := informerFactory.Rbac().V1().RoleBindings()

	ah := &admissionHandler{
		lister:        informer.Lister(),
		kubeClient:    kubeClientSet,
		dynamicClient: dynamicClient,
	}

	go informerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, informer.Informer().HasSynced); !ok {
		klog.Errorf("failed to wait for kubernetes caches to sync")
		return fmt.Errorf("failed to wait for kubernetes caches to sync")
	}

	http.HandleFunc("/mutating", ah.serveMutateResource)
	http.HandleFunc("/validating", ah.serverValidateResource)

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
