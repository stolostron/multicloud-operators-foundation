// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package options

import (
	"reflect"
	"testing"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewStorageSerializationOptions(t *testing.T) {
	tests := []struct {
		name string
		want *StorageSerializationOptions
	}{

		{"case1:", &StorageSerializationOptions{StorageVersions: "clusterregistry.k8s.io/v1alpha1,mcm.ibm.com/v1alpha1,v1", DefaultStorageVersions: "clusterregistry.k8s.io/v1alpha1,mcm.ibm.com/v1alpha1,v1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewStorageSerializationOptions(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStorageSerializationOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStorageSerializationOptions_StorageGroupsToEncodingVersion(t *testing.T) {
	//groupVersion1 := schema.GroupVersion{"group1", "v1"}
	groupVersion2 := schema.GroupVersion{"", "v2"}
	type fields struct {
		StorageVersions        string
		DefaultStorageVersions string
	}
	tests := []struct {
		name    string
		fields  fields
		want    map[string]schema.GroupVersion
		wantErr bool
	}{
		{"case1:", fields{StorageVersions: "v2", DefaultStorageVersions: "v1"}, map[string]schema.GroupVersion{"": groupVersion2}, false},
		{"case2:", fields{}, map[string]schema.GroupVersion{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &StorageSerializationOptions{
				StorageVersions:        tt.fields.StorageVersions,
				DefaultStorageVersions: tt.fields.DefaultStorageVersions,
			}
			got, err := s.StorageGroupsToEncodingVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("StorageSerializationOptions.StorageGroupsToEncodingVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StorageSerializationOptions.StorageGroupsToEncodingVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStorageSerializationOptions_AddFlags(t *testing.T) {
	type fields struct {
		StorageVersions        string
		DefaultStorageVersions string
	}
	type args struct {
		fs *pflag.FlagSet
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"case1:", fields{"v2", "v1"}, args{&pflag.FlagSet{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &StorageSerializationOptions{
				StorageVersions:        tt.fields.StorageVersions,
				DefaultStorageVersions: tt.fields.DefaultStorageVersions,
			}
			s.AddFlags(tt.args.fs)
		})
	}
}
