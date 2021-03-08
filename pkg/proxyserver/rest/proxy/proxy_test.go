package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/getter"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func Test_getSubResource(t *testing.T) {
	pathTestCases := []struct {
		path        string
		subResource string
	}{
		{
			path:        "/",
			subResource: "",
		},
		{
			path:        "/apis/proxy.open-cluster-management.io/v1beta1/clusterstatuses/cluster1/aggregator/sync/",
			subResource: "",
		},
		{
			path:        "/apis/proxy.open-cluster-management.io/v1beta1/namespaces/cluster1/clusterstatuses/cluster1/aggregator/sync/",
			subResource: "sync",
		},
	}

	for _, testCase := range pathTestCases {
		if subResouce, _ := getSubResource(testCase.path); subResouce != testCase.subResource {
			t.Errorf("get subResource fail %s", testCase.path)
		}
	}
}

type restRes struct{}

func (rp *restRes) Object(statusCode int, obj runtime.Object) {
}

func (rp *restRes) Error(err error) {
}

func Test_ServeHTTP(t *testing.T) {
	rest := NewProxyRest(getter.NewProxyServiceInfoGetter())
	obj, flag, path := rest.NewConnectOptions()
	if obj == nil || (flag && path == "") {
		t.Errorf("rest NewConnectOptions test case failed")
	}

	if methods := rest.ConnectMethods(); len(methods) == 0 {
		t.Errorf("rest ConnectMethods test case failed")
	}

	opt := rest.New()
	response := &restRes{}
	reqCtx := request.WithRequestInfo(context.Background(), &request.RequestInfo{})
	handler, _ := rest.Connect(reqCtx, "cluster0", opt, response)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := r.WithContext(reqCtx)
		handler.ServeHTTP(w, req)
	}))
	defer s.Close()
	res, err := http.Get(s.URL + "/apis/proxy.open-cluster-management.io/v1beta1/namespaces/cluster1/clusterstatuses/cluster1/aggregator/sync")
	if err != nil {
		t.Errorf("test ServerHTTP failed")
	}
	if res.StatusCode == http.StatusOK {
		t.Errorf("test ServerHTTP failed")
	}
	res.Body.Close()
}
