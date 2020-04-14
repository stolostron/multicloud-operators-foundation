// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rest

import (
	"context"
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	klusterlet "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
)

type connInfoGetter struct{}

func (s *connInfoGetter) GetConnectionInfo(ctx context.Context, clusterName string) (*klusterlet.ConnectionInfo, error) {
	return &klusterlet.ConnectionInfo{
		Scheme:   "schema",
		Hostname: "hostname",
		IP:       "127.0.0.1",
	}, nil
}

type restOptGetter struct{}

func (r *restOptGetter) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	return generic.RESTOptions{}, nil
}

type restRes struct{}

func (rp *restRes) Object(statusCode int, obj runtime.Object) {
}
func (rp *restRes) Error(err error) {
}

func TestNewLogRest(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	logRest := NewLogRest(&restOptGetter{}, &connInfoGetter{})
	logRest.New()
	logRest.ConnectMethods()
	logRest.NewConnectOptions()
	_, err := logRest.Connect(ctx, "id", &mcm.ClusterRestOptions{}, &restRes{})
	if err == nil {
		t.Errorf("Should have error")
	}
	logRest.NewGetOptions()
}

func TestNewMonitorRest(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	monitorRest := NewMonitorRest(&restOptGetter{}, &connInfoGetter{})
	monitorRest.New()
	monitorRest.ConnectMethods()
	monitorRest.NewConnectOptions()
	_, err := monitorRest.Connect(ctx, "id", &mcm.ClusterRestOptions{}, &restRes{})
	if err != nil {
		t.Errorf("Error,%s", err)
	}
	monitorRest.NewGetOptions()
}
