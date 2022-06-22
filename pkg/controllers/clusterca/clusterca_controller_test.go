package clusterca

import (
	"reflect"
	"testing"

	clusterinfov1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func Test_updateClientConfig(t *testing.T) {
	tests := []struct {
		name              string
		clusterConfig     []clusterv1.ClientConfig
		clusterinfoConfig clusterinfov1beta1.ClientConfig
		wantConfig        []clusterv1.ClientConfig
		wantUpdate        bool
	}{
		{
			name:              "all nil",
			clusterConfig:     []clusterv1.ClientConfig{},
			clusterinfoConfig: clusterinfov1beta1.ClientConfig{},
			wantConfig:        []clusterv1.ClientConfig{},
			wantUpdate:        false,
		},
		{
			name:          "clusterconfig is null",
			clusterConfig: []clusterv1.ClientConfig{},
			clusterinfoConfig: clusterinfov1beta1.ClientConfig{
				URL:      "https:url.com:443",
				CABundle: []byte("ca data"),
			},
			wantConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:url.com:443",
					CABundle: []byte("ca data"),
				},
			},
			wantUpdate: true,
		},
		{
			name: "clusterinfoconfig is null",
			clusterConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:url.com:443",
					CABundle: []byte("ca data"),
				},
			},
			clusterinfoConfig: clusterinfov1beta1.ClientConfig{},
			wantConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:url.com:443",
					CABundle: []byte("ca data"),
				},
			},
			wantUpdate: false,
		},
		{
			name: "both of them is not null",
			clusterConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:clusterurl.com:443",
					CABundle: []byte("ca data"),
				},
			},
			clusterinfoConfig: clusterinfov1beta1.ClientConfig{
				URL:      "https:infourl.com:443",
				CABundle: []byte("ca data"),
			},
			wantConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:clusterurl.com:443",
					CABundle: []byte("ca data"),
				},
				{
					URL:      "https:infourl.com:443",
					CABundle: []byte("ca data"),
				},
			},
			wantUpdate: true,
		},
		{
			name: "update cluster config",
			clusterConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:clusterurl.com:443",
					CABundle: []byte("ca data"),
				},
				{
					URL:      "https:infourl.com:443",
					CABundle: []byte("info data"),
				},
			},
			clusterinfoConfig: clusterinfov1beta1.ClientConfig{
				URL:      "https:infourl.com:443",
				CABundle: []byte("new info data"),
			},
			wantConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:clusterurl.com:443",
					CABundle: []byte("ca data"),
				},
				{
					URL:      "https:infourl.com:443",
					CABundle: []byte("new info data"),
				},
			},
			wantUpdate: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			returnConfig, needUpdate := updateClientConfig(test.clusterConfig, test.clusterinfoConfig)
			if needUpdate != test.wantUpdate {
				t.Errorf("case: %v, needupdate is: %v, wantUpdate is : %v", test.name, needUpdate, test.wantUpdate)
				return
			}
			if !reflect.DeepEqual(returnConfig, test.wantConfig) {
				t.Errorf("case:%v, returnConfig:%v, wantConfig:%v.", test.name, returnConfig, test.wantConfig)
			}
			return
		})
	}
}
