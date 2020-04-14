// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rest

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/aggregator"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestNewAggregateRest(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	kubeclient := kubefake.NewSimpleClientset()
	infoGetter := aggregator.NewInfoGetters(kubeclient)
	aggRest := NewAggregateRest(infoGetter)
	aggRest.New()
	aggRest.ConnectMethods()
	aggRest.NewConnectOptions()
	clusterResOpt := mcm.ClusterRestOptions{}
	aggRest.Connect(ctx, "id1", &clusterResOpt, &restRes{})
}

func Test_getPath(t *testing.T) {
	type args struct {
		requestPath string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"case1", args{requestPath: "a/b/c"}, ""},
		{"case1", args{requestPath: ""}, ""},
		{"case1", args{requestPath: "1/2/3/4/5/6/7/8/9/0"}, "9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPath(tt.args.requestPath); got != tt.want {
				t.Errorf("getPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
