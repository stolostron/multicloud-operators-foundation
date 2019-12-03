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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	utilflag "k8s.io/apiserver/pkg/util/flag"
)

// NewStorageFactory builds the DefaultStorageFactory.
// Merges defaultResourceConfig with the user specified overrides and merges
// defaultAPIResourceConfig with the corresponding user specified overrides as well.
func NewStorageFactory(
	storageConfig storagebackend.Config,
	defaultMediaType string,
	serializer runtime.StorageSerializer,
	defaultResourceEncoding *serverstorage.DefaultResourceEncodingConfig,
	storageEncodingOverrides map[string]schema.GroupVersion,
	resourceEncodingOverrides []schema.GroupVersionResource,
	defaultAPIResourceConfig *serverstorage.ResourceConfig,
	resourceConfigOverrides utilflag.ConfigurationMap) (*serverstorage.DefaultStorageFactory, error) {
	resourceEncodingConfig := mergeGroupEncodingConfigs(defaultResourceEncoding, storageEncodingOverrides)
	resourceEncodingConfig = mergeResourceEncodingConfigs(resourceEncodingConfig, resourceEncodingOverrides)
	apiResourceConfig, err := mergeAPIResourceConfigs(defaultAPIResourceConfig, resourceConfigOverrides)
	if err != nil {
		return nil, err
	}
	return serverstorage.NewDefaultStorageFactory(
		storageConfig, defaultMediaType, serializer, resourceEncodingConfig, apiResourceConfig, nil), nil
}

// Merges the given defaultResourceConfig with specifc GroupvVersionResource overrides.
func mergeResourceEncodingConfigs(
	defaultResourceEncoding *serverstorage.DefaultResourceEncodingConfig,
	resourceEncodingOverrides []schema.GroupVersionResource) *serverstorage.DefaultResourceEncodingConfig {
	resourceEncodingConfig := defaultResourceEncoding
	for _, gvr := range resourceEncodingOverrides {
		resourceEncodingConfig.SetResourceEncoding(gvr.GroupResource(), gvr.GroupVersion(),
			schema.GroupVersion{Group: gvr.Group, Version: runtime.APIVersionInternal})
	}
	return resourceEncodingConfig
}

// Merges the given defaultResourceConfig with specifc GroupVersion overrides.
func mergeGroupEncodingConfigs(
	defaultResourceEncoding *serverstorage.DefaultResourceEncodingConfig,
	storageEncodingOverrides map[string]schema.GroupVersion) *serverstorage.DefaultResourceEncodingConfig {
	resourceEncodingConfig := defaultResourceEncoding
	for group, storageEncodingVersion := range storageEncodingOverrides {
		resourceEncodingConfig.SetVersionEncoding(
			group, storageEncodingVersion, schema.GroupVersion{Group: group, Version: runtime.APIVersionInternal})
	}
	return resourceEncodingConfig
}

// Merges the given defaultAPIResourceConfig with the given resourceConfigOverrides.
func mergeAPIResourceConfigs(
	defaultAPIResourceConfig *serverstorage.ResourceConfig,
	resourceConfigOverrides utilflag.ConfigurationMap) (*serverstorage.ResourceConfig, error) {
	resourceConfig := defaultAPIResourceConfig
	overrides := resourceConfigOverrides

	// "api/legacy=false" allows users to disable legacy api versions.
	disableLegacyAPIs := false
	legacyAPIFlagValue, ok := overrides["api/legacy"]
	if ok && legacyAPIFlagValue == "false" {
		disableLegacyAPIs = true
	}
	_ = disableLegacyAPIs // hush the compiler while we don't have legacy APIs to disable.

	// "<resourceSpecifier>={true|false} allows users to enable/disable API.
	// This takes preference over api/all and api/legacy, if specified.
	// Iterate through all group/version overrides specified in runtimeConfig.
	for key := range overrides {
		if key == "api/all" || key == "api/legacy" {
			// Have already handled them above. Can skip them here.
			continue
		}
		tokens := strings.Split(key, "/")
		if len(tokens) != 2 {
			continue
		}
		groupVersionString := tokens[0] + "/" + tokens[1]
		// HACK: Hack for "v1" legacy group version.
		// Remove when we stop supporting the legacy group version.
		if groupVersionString == "api/v1" {
			groupVersionString = "v1"
		}
		_, err := schema.ParseGroupVersion(groupVersionString)
		if err != nil {
			return nil, fmt.Errorf("invalid key %s", key)
		}
	}

	// Iterate through all group/version/resource overrides specified in runtimeConfig.
	for key := range overrides {
		tokens := strings.Split(key, "/")
		if len(tokens) != 3 {
			continue
		}
		groupVersionString := tokens[0] + "/" + tokens[1]
		// HACK: Hack for "v1" legacy group version.
		// Remove when we stop supporting the legacy group version.
		if groupVersionString == "api/v1" {
			groupVersionString = "v1"
		}
		groupVersion, err := schema.ParseGroupVersion(groupVersionString)
		if err != nil {
			return nil, fmt.Errorf("invalid key %s", key)
		}

		if !resourceConfig.AnyVersionForGroupEnabled(groupVersionString) {
			return nil, fmt.Errorf("%v is disabled, you cannot configure its resources individually", groupVersion)
		}
	}
	return resourceConfig, nil
}
