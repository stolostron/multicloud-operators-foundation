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
package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	clusterregistry "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/clusterregistry/install"
	hcm "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/install"
)

var (
	// Scheme for API object types
	Scheme = runtime.NewScheme()
	// ParameterCodec handles versioning of objects that are converted to query parameters.
	ParameterCodec = runtime.NewParameterCodec(Scheme)
	// Codecs for creating a server config
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	hcm.Install(Scheme)
	clusterregistry.Install(Scheme)

	// we need to add the options to empty v1
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
		&metav1.ExportOptions{},
		&metav1.WatchEvent{},
		&metav1beta1.Table{},
	)
}
