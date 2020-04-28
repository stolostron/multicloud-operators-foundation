package getter

import (
	"reflect"
	"sync"

	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

type ProxyServiceInfo struct {
	Name             string
	SubResource      string
	ServiceName      string
	ServiceNamespace string
	ServicePort      string
	RootPath         string
	UseID            bool
	RestConfig       *rest.Config
}

type ProxyServiceInfoGetter struct {
	mutex sync.RWMutex
	// key is sub resource name
	proxyServiceInfos map[string]*ProxyServiceInfo
}

func NewProxyServiceInfoGetter() *ProxyServiceInfoGetter {
	return &ProxyServiceInfoGetter{
		proxyServiceInfos: make(map[string]*ProxyServiceInfo),
	}
}

func (g *ProxyServiceInfoGetter) Get(subResource string) *ProxyServiceInfo {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return g.proxyServiceInfos[subResource]
}

func (g *ProxyServiceInfoGetter) Add(proxyServiceInfo *ProxyServiceInfo) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if old, existed := g.proxyServiceInfos[proxyServiceInfo.SubResource]; existed {
		if !reflect.DeepEqual(old, proxyServiceInfo) {
			if old.Name != proxyServiceInfo.Name {
				klog.Errorf("The proxy service configMap %s cannot be updated by %s", old.Name, proxyServiceInfo.Name)
				return
			}

			klog.Infof("Update proxy service info %s", proxyServiceInfo.Name)
			g.proxyServiceInfos[proxyServiceInfo.SubResource] = proxyServiceInfo
		}
		return
	}

	klog.Infof("Add proxy service info %s", proxyServiceInfo.Name)
	g.proxyServiceInfos[proxyServiceInfo.SubResource] = proxyServiceInfo
}

func (g *ProxyServiceInfoGetter) Delete(proxyServiceInfoName string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for key, serviceInfo := range g.proxyServiceInfos {
		if serviceInfo.Name == proxyServiceInfoName {
			klog.Infof("Delete proxy service info %s", proxyServiceInfoName)
			delete(g.proxyServiceInfos, key)
			break
		}
	}
}
