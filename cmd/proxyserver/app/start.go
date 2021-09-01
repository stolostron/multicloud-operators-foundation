// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"strings"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/proxyserver/app/options"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/controller"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/getter"
	apilabels "k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
)

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

	dynamicClient, err := dynamic.NewForConfig(clusterCfg)
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

	informerFactory := informers.NewSharedInformerFactory(kubeClient, 10*time.Minute)
	proxyGetter := getter.NewProxyServiceInfoGetter()
	ctrl := controller.NewProxyServiceInfoController(kubeClient, configMapLabels, informerFactory, proxyGetter, stopCh)

	apiServerConfig, err := s.APIServerConfig()
	if err != nil {
		return err
	}

	logGetter, err := getter.NewLogConnectionInfoGetter(s.ClientOptions.Config(dynamicClient))
	if err != nil {
		return nil
	}

	proxyServer, err := NewProxyServer(clusterClient, informerFactory, clusterInformers, apiServerConfig, proxyGetter, logGetter)
	if err != nil {
		return err
	}

	go ctrl.Run()
	clusterInformers.Start(stopCh)
	informerFactory.Start(stopCh)

	return proxyServer.Run(stopCh)
}
