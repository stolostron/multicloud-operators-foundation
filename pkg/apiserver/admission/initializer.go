// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package admission

import (
	"k8s.io/apiserver/pkg/admission"

	"k8s.io/apiserver/pkg/admission/initializer"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/internalclientset"
	clusterclient "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset"
	clusterinformers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	informers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/internalversion"
)

// WantsInternalHCMClientSet defines a function which sets ClientSet for admission plugins that need it
type WantsInternalHCMClientSet interface {
	SetInternalHCMClientSet(internalclientset.Interface)
	admission.InitializationValidator
}

// WantsInternalHCMInformerFactory defines a function which sets InformerFactory for admission plugins that need it
type WantsInternalHCMInformerFactory interface {
	SetInternalHCMInformerFactory(informers.SharedInformerFactory)
	admission.InitializationValidator
}

// WantsInternalClusterClient defines a function which sets ClientSet for admission plugins that need it
type WantsInternalClusterClient interface {
	SetInternalClusterClientSet(clusterclient.Interface)
	admission.InitializationValidator
}

// WantsInternalClusterInformerFactory defines a function which sets InformerFactory for admission plugins that need it
type WantsInternalClusterInformerFactory interface {
	SetInternalClusterInformerFactory(clusterinformers.SharedInformerFactory)
	admission.InitializationValidator
}

type pluginInitializer struct {
	internalClient internalclientset.Interface
	informers      informers.SharedInformerFactory

	kubeClient    kubeclientset.Interface
	kubeInformers kubeinformers.SharedInformerFactory

	clusterClient    clusterclient.Interface
	clusterInformers clusterinformers.SharedInformerFactory
}

var _ admission.PluginInitializer = pluginInitializer{}

// NewPluginInitializer constructs new instance of PluginInitializer
func NewPluginInitializer(
	internalClient internalclientset.Interface, sharedInformers informers.SharedInformerFactory,
	kubeClient kubeclientset.Interface, kubeInformers kubeinformers.SharedInformerFactory,
	clusterClient clusterclient.Interface, clusterInformers clusterinformers.SharedInformerFactory) admission.PluginInitializer {
	return pluginInitializer{
		internalClient:   internalClient,
		informers:        sharedInformers,
		kubeClient:       kubeClient,
		kubeInformers:    kubeInformers,
		clusterClient:    clusterClient,
		clusterInformers: clusterInformers,
	}
}

// Initialize checks the initialization interfaces implemented by each plugin
// and provide the appropriate initialization data
func (i pluginInitializer) Initialize(plugin admission.Interface) {
	if wants, ok := plugin.(WantsInternalHCMClientSet); ok {
		wants.SetInternalHCMClientSet(i.internalClient)
	}

	if wants, ok := plugin.(WantsInternalHCMInformerFactory); ok {
		wants.SetInternalHCMInformerFactory(i.informers)
	}

	if wants, ok := plugin.(initializer.WantsExternalKubeClientSet); ok {
		wants.SetExternalKubeClientSet(i.kubeClient)
	}

	if wants, ok := plugin.(initializer.WantsExternalKubeInformerFactory); ok {
		wants.SetExternalKubeInformerFactory(i.kubeInformers)
	}

	if wants, ok := plugin.(WantsInternalClusterClient); ok {
		wants.SetInternalClusterClientSet(i.clusterClient)
	}

	if wants, ok := plugin.(WantsInternalClusterInformerFactory); ok {
		wants.SetInternalClusterInformerFactory(i.clusterInformers)
	}
}
