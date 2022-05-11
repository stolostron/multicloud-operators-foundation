//
// Copyright 2022 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package filteredcache

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
)

func getClientForGVK(gvk schema.GroupVersionKind, config *rest.Config, scheme *runtime.Scheme) (toolscache.Getter, error) {
	// Create a client for fetching resources
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	switch gvk.GroupVersion() {
	case corev1.SchemeGroupVersion:
		return k8sClient.CoreV1().RESTClient(), nil
	case appsv1.SchemeGroupVersion:
		return k8sClient.AppsV1().RESTClient(), nil
	case batchv1.SchemeGroupVersion:
		return k8sClient.BatchV1().RESTClient(), nil
	case networkingv1.SchemeGroupVersion:
		return k8sClient.NetworkingV1().RESTClient(), nil
	case rbacv1.SchemeGroupVersion:
		return k8sClient.RbacV1().RESTClient(), nil
	case storagev1.SchemeGroupVersion:
		return k8sClient.StorageV1().RESTClient(), nil
	case certificatesv1beta1.SchemeGroupVersion:
		return k8sClient.CertificatesV1beta1().RESTClient(), nil
	default:
		gv := gvk.GroupVersion()
		cfg := rest.CopyConfig(config)
		cfg.GroupVersion = &gv
		if gvk.Group == "" {
			cfg.APIPath = "/api"
		} else {
			cfg.APIPath = "/apis"
		}
		if cfg.UserAgent == "" {
			cfg.UserAgent = rest.DefaultKubernetesUserAgent()
		}
		if cfg.NegotiatedSerializer == nil {
			cfg.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}
		}
		return rest.RESTClientFor(cfg)
	}
}
