// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"fmt"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"strings"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/cmd/proxyserver/app/options"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/controller"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/getter"
	apilabels "k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
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

	logSecretInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, 10*time.Minute, informers.WithTweakListOptions(
		func(listOptions *metav1.ListOptions) {
			listOptions.FieldSelector = fields.OneTermEqualSelector("metadata.name", helpers.LogManagedServiceAccountName).String()
		}))

	componentNs, err := utils.GetComponentNamespace()
	if err != nil {
		return err
	}
	proxyServiceHost := fmt.Sprintf("%s.%s.svc:%s", s.ClientOptions.ProxyServiceName, componentNs, s.ClientOptions.ProxyServicePort)
	logProxyGetter := getter.NewLogProxyGetter(logSecretInformerFactory.Core().V1().Secrets().Lister(),
		proxyServiceHost, s.ClientOptions.ProxyServiceCAFile)

	proxyServer, err := NewProxyServer(clusterClient, informerFactory, clusterInformers, apiServerConfig,
		proxyGetter, logProxyGetter)
	if err != nil {
		return err
	}

	go ctrl.Run()
	clusterInformers.Start(stopCh)
	informerFactory.Start(stopCh)
	configmapInformerFactory.Start(stopCh)
	logSecretInformerFactory.Start(stopCh)
	return proxyServer.Run(stopCh)
}
