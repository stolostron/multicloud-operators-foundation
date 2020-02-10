// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package apiserver

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
)

func TestDefaultAPIResourceConfigSource(t *testing.T) {
	var rc1 = map[schema.GroupVersion]bool{{"mcm.ibm.com", "v1beta1"}: true, {"mcm.ibm.com", "v1alpha1"}: true, {"clusterregistry.k8s.io", "v1alpha1"}: true}

	tests := []struct {
		name string
		want *serverstorage.ResourceConfig
	}{
		{"case1:", &serverstorage.ResourceConfig{rc1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultAPIResourceConfigSource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultAPIResourceConfigSource() = %v, want %v", got, tt.want)
			}
		})
	}
}
