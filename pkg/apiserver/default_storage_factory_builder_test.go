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

package apiserver

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	cliflag "k8s.io/component-base/cli/flag"
)

func TestNewStorageFactory(t *testing.T) {
	storageConfig := storagebackend.Config{}
	defaultResourceEncoding := &serverstorage.DefaultResourceEncodingConfig{}
	resourceEncodingOverrides := []schema.GroupVersionResource{}
	storageEncodingOverrides := map[string]schema.GroupVersion{}
	resourceEncodingConfig := mergeResourceEncodingConfigs(defaultResourceEncoding, resourceEncodingOverrides)
	if resourceEncodingConfig == nil {
		t.Errorf("fake test running failed")
	}

	defaultAPIResourceConfig := &serverstorage.ResourceConfig{}
	resourceConfigOverrides := cliflag.ConfigurationMap{}
	_, err := NewStorageFactory(
		storageConfig, "", nil, defaultResourceEncoding, storageEncodingOverrides,
		resourceEncodingOverrides, defaultAPIResourceConfig, resourceConfigOverrides)
	if err != nil {
		t.Errorf("running failed")
	}
}

func Test_mergeAPIResourceConfigs(t *testing.T) {
	g1v1 := schema.GroupVersion{Group: "group1", Version: "version1"}
	g1v2 := schema.GroupVersion{Group: "group1", Version: "version2"}
	g2v1 := schema.GroupVersion{Group: "group2", Version: "api/v1"}
	groupVersionConfigs := map[schema.GroupVersion]bool{g1v1: true, g1v2: true, g2v1: false}
	defaultAPIResourceConfig := &serverstorage.ResourceConfig{GroupVersionConfigs: groupVersionConfigs}

	resourceConfigOverrides := cliflag.ConfigurationMap{"conf1": "val1", "api/all": "val2"}

	want := &serverstorage.ResourceConfig{GroupVersionConfigs: map[schema.GroupVersion]bool{g1v1: true, g1v2: true, g2v1: false}}
	wantErr := false
	result, err := mergeAPIResourceConfigs(defaultAPIResourceConfig, resourceConfigOverrides)
	if (err != nil) != wantErr {
		t.Errorf("mergeAPIResourceConfigs() error = %v, wantErr %v", err, wantErr)
		return
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("mergeAPIResourceConfigs() = %v, want %v ", result, want)
	}

	resourceConfigOverrides1 := cliflag.ConfigurationMap{"api/all/": "val3", "api/v1": "vvv1"}
	_, err1 := mergeAPIResourceConfigs(defaultAPIResourceConfig, resourceConfigOverrides1)
	if err1 == nil {
		t.Errorf("Coverred that interface have disabled")
	}
}
