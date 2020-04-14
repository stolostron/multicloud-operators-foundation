// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/aggregator/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
)

type restOptGetter struct{}

func (r *restOptGetter) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	return generic.RESTOptions{}, nil
}

type restRes struct{}

func (rp *restRes) Object(statusCode int, obj runtime.Object) {
}
func (rp *restRes) Error(err error) {
}
func TestNewAggregateRest(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()

	conInfoGetter := v1alpha1.ConnectionInfoGetter{}
	aggRest := NewAggregateRest(&restOptGetter{}, &conInfoGetter)
	aggRest.New()
	aggRest.ConnectMethods()
	aggRest.NewConnectOptions()
	clusterResOpt := mcm.ClusterRestOptions{}
	aggRest.Connect(ctx, "id1", &clusterResOpt, &restRes{})
}
