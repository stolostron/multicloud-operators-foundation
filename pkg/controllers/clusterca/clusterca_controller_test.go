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
		lastAppliedURL    string
		wantConfig        []clusterv1.ClientConfig
		wantUpdateMC      bool
		wantUpdateMCI     bool
	}{
		{
			name:              "all nil",
			clusterConfig:     []clusterv1.ClientConfig{},
			clusterinfoConfig: clusterinfov1beta1.ClientConfig{},
			wantConfig:        []clusterv1.ClientConfig{},
			wantUpdateMC:      false,
			wantUpdateMCI:     false,
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
			wantUpdateMC:  true,
			wantUpdateMCI: true,
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
			wantUpdateMC:  false,
			wantUpdateMCI: false,
		},
		{
			name: "both of them is not null, and order matters",
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
					URL:      "https:infourl.com:443",
					CABundle: []byte("ca data"),
				},
				{
					URL:      "https:clusterurl.com:443",
					CABundle: []byte("ca data"),
				},
			},
			wantUpdateMC:  true,
			wantUpdateMCI: true,
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
			wantUpdateMC:  true,
			wantUpdateMCI: false,
		},
		{
			name: "replace the last applied url with new url",
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
			lastAppliedURL: "https:clusterurl.com:443",
			wantConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:infourl.com:443",
					CABundle: []byte("ca data"),
				},
			},
			wantUpdateMC:  true,
			wantUpdateMCI: true,
		},
		{
			name: "replace the last applied url with new url, order not change",
			clusterConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:clusterurl-1.com:443",
					CABundle: []byte("new info data"),
				},
				{
					URL:      "https:clusterurl.com:443",
					CABundle: []byte("ca data"),
				},
				{
					URL:      "https:clusterurl-2.com:443",
					CABundle: []byte("new info data"),
				},
			},
			clusterinfoConfig: clusterinfov1beta1.ClientConfig{
				URL:      "https:infourl.com:443",
				CABundle: []byte("ca data"),
			},
			lastAppliedURL: "https:clusterurl.com:443",
			wantConfig: []clusterv1.ClientConfig{
				{
					URL:      "https:clusterurl-1.com:443",
					CABundle: []byte("new info data"),
				},
				{
					URL:      "https:infourl.com:443",
					CABundle: []byte("ca data"),
				},
				{
					URL:      "https:clusterurl-2.com:443",
					CABundle: []byte("new info data"),
				},
			},
			wantUpdateMC:  true,
			wantUpdateMCI: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			returnConfig, needUpdateMC, needUpdateMCI := updateClientConfig(
				test.clusterConfig, test.clusterinfoConfig, test.lastAppliedURL)
			if needUpdateMC != test.wantUpdateMC {
				t.Errorf("case: %v, expected update managed cluster: %v, but got : %v",
					test.name, test.wantUpdateMC, needUpdateMC)
				return
			}
			if needUpdateMCI != test.wantUpdateMCI {
				t.Errorf("case: %v, expected update managed cluster info: %v, but got : %v",
					test.name, test.wantUpdateMCI, needUpdateMCI)
				return
			}
			if !reflect.DeepEqual(returnConfig, test.wantConfig) {
				t.Errorf("case:%v, returnConfig:%v, wantConfig:%v.", test.name, returnConfig, test.wantConfig)
			}
			return
		})
	}
}
