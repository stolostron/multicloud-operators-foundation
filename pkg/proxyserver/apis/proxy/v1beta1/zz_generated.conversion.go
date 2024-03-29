//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright (c) 2020 Red Hat, Inc.

// Code generated by conversion-gen. DO NOT EDIT.

package v1beta1

import (
	url "net/url"

	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*url.Values)(nil), (*ClusterStatusProxyOptions)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_url_Values_To_v1beta1_ClusterStatusProxyOptions(a.(*url.Values), b.(*ClusterStatusProxyOptions), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_url_Values_To_v1beta1_ClusterStatusProxyOptions(in *url.Values, out *ClusterStatusProxyOptions, s conversion.Scope) error {
	// WARNING: Field TypeMeta does not have json tag, skipping.

	if values, ok := map[string][]string(*in)["path"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_string(&values, &out.Path, s); err != nil {
			return err
		}
	} else {
		out.Path = ""
	}
	return nil
}

// Convert_url_Values_To_v1beta1_ClusterStatusProxyOptions is an autogenerated conversion function.
func Convert_url_Values_To_v1beta1_ClusterStatusProxyOptions(in *url.Values, out *ClusterStatusProxyOptions, s conversion.Scope) error {
	return autoConvert_url_Values_To_v1beta1_ClusterStatusProxyOptions(in, out, s)
}
