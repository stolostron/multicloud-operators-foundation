package getter

import (
	"testing"

	"k8s.io/client-go/rest"
)

func TestProxyServiceInfoGetter(t *testing.T) {
	serviceInfo1 := &ProxyServiceInfo{
		Name:        "default/search",
		SubResource: "sync",
		UseID:       true,
		RestConfig:  &rest.Config{},
	}
	serviceInfo2 := &ProxyServiceInfo{
		Name:        "default/search",
		SubResource: "sync",
		UseID:       false,
		RestConfig:  &rest.Config{},
	}
	serviceInfo3 := &ProxyServiceInfo{
		Name:        "kube-system/search",
		SubResource: "sync",
		UseID:       false,
		RestConfig:  &rest.Config{},
	}

	getter := NewProxyServiceInfoGetter()

	getter.Add(serviceInfo1)
	rst := getter.Get(serviceInfo1.SubResource)
	if rst == nil {
		t.Errorf("getter Add/Get test case fails")
	} else if !rst.UseID {
		t.Errorf("getter Add/Get test case fails")
	}

	getter.Add(serviceInfo3)
	rst = getter.Get(serviceInfo3.SubResource)
	if rst == nil {
		t.Errorf("getter update/Get test case 1 fails")
	} else if !rst.UseID {
		t.Errorf("getter update/Get test case 1 fails")
	}

	getter.Add(serviceInfo2)
	rst = getter.Get(serviceInfo2.SubResource)
	if rst == nil {
		t.Errorf("getter update/Get test case 2 fails")
	} else if rst.UseID {
		t.Errorf("getter update/Get test case 2 fails")
	}

	getter.Delete(serviceInfo2.Name)
	rst = getter.Get(serviceInfo2.SubResource)
	if rst != nil {
		t.Errorf("getter delete test case fails")
	}
}
