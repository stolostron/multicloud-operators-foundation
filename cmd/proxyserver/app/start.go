// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stolostron/multicloud-operators-foundation/cmd/proxyserver/app/options"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/controller"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/getter"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apilabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	addonv1alpha1 "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
)

const clusterPermissionCRDName = "clusterpermissions.rbac.open-cluster-management.io"

func Run(s *options.Options, stopCh <-chan struct{}) error {
	if err := s.SetDefaults(); err != nil {
		return err
	}

	if errs := s.Validate(); len(errs) != 0 {
		return utilerrors.NewAggregate(errs)
	}

	clusterCfg, err := clientcmd.BuildConfigFromFlags("", s.KubeConfigFile)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}

	configMapLabels, err := apilabels.ConvertSelectorToLabelsMap(strings.TrimSuffix(s.ConfigMapLabels, ","))
	if err != nil {
		return err
	}

	clusterClient, err := clusterv1client.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}

	clusterInformers := clusterv1informers.NewSharedInformerFactory(clusterClient, 10*time.Minute)

	addonClient, err := addonv1alpha1.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, 10*time.Minute)
	proxyGetter := getter.NewProxyServiceInfoGetter()
	configmapInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, 10*time.Minute, informers.WithTweakListOptions(
		func(listOptions *metav1.ListOptions) {
			matchExpressions := []metav1.LabelSelectorRequirement{}
			for key, value := range configMapLabels {
				matchExpressions = append(matchExpressions,
					metav1.LabelSelectorRequirement{
						Key:      key,
						Values:   []string{value},
						Operator: metav1.LabelSelectorOpIn,
					})
			}
			selector := &metav1.LabelSelector{MatchExpressions: matchExpressions}
			listOptions.LabelSelector = metav1.FormatLabelSelector(selector)
		}))
	ctrl := controller.NewProxyServiceInfoController(kubeClient, configMapLabels,
		configmapInformerFactory.Core().V1().ConfigMaps(), proxyGetter, stopCh)

	apiServerConfig, err := s.APIServerConfig()
	if err != nil {
		return err
	}

	componentNs, err := utils.GetComponentNamespace()
	if err != nil {
		return err
	}
	proxyServiceHost := fmt.Sprintf("%s.%s.svc:%s", s.ClientOptions.ProxyServiceName, componentNs, s.ClientOptions.ProxyServicePort)

	logProxyGetter := getter.NewLogProxyGetter(addonClient, kubeClient, proxyServiceHost, s.ClientOptions.ProxyServiceCAFile)

	apiextensionsClient, err := apiextensionsclient.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}
	clusterPermissionClient, err := dynamic.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}
	clusterPermissionFactory := dynamicinformer.NewDynamicSharedInformerFactory(clusterPermissionClient, 10*time.Minute)
	clusterPermissionInformers := clusterPermissionFactory.ForResource(schema.GroupVersionResource{
		Group:    "rbac.open-cluster-management.io",
		Version:  "v1alpha1",
		Resource: "clusterpermissions",
	})
	clusterPermissionLister := clusterPermissionInformers.Lister()

	proxyServer, err := NewProxyServer(clusterClient, informerFactory, clusterInformers,
		clusterPermissionInformers.Informer(), clusterPermissionLister,
		apiServerConfig, proxyGetter, logProxyGetter)
	if err != nil {
		return err
	}

	go ctrl.Run()
	clusterInformers.Start(stopCh)
	informerFactory.Start(stopCh)
	configmapInformerFactory.Start(stopCh)

	// Only start clusterpermission informer when clusterpermission crd installed
	go func() {
		if err := wait.PollUntilContextCancel(context.Background(), 60*time.Second, true, func(ctx context.Context) (bool, error) {
			_, err := apiextensionsClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, clusterPermissionCRDName, metav1.GetOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					klog.Errorf("failed to get crd %s, %v", clusterPermissionCRDName, err)
				}

				return false, nil
			}

			klog.Infof("Starting ClusterPermission Informer")
			clusterPermissionFactory.Start(stopCh)
			return true, nil
		}); err != nil {
			klog.Errorf("failed to check clusterpermission crd %v", err)
		}
	}()

	return proxyServer.Run(stopCh)
}
