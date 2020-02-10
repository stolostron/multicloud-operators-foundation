// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mongo

import (
	"context"
	"fmt"
	"net/http"

	mcmv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage/mongo/weave"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
)

type TopologyRest struct {
	inserter *weave.ClusterTopologyInserter
}

func NewTopologyRest(optsGetter generic.RESTOptionsGetter, inserter *weave.ClusterTopologyInserter) *TopologyRest {
	return &TopologyRest{inserter: inserter}
}

// ProducesMIMETypes returns a list of the MIME types the specified HTTP verb (GET, POST, DELETE,
// PATCH) can respond with.
func (t *TopologyRest) ProducesMIMETypes(verb string) []string {
	return []string{"application/json"}
}

// ProducesObject returns an object the specified HTTP verb respond with. It will overwrite storage object if
// it is not nil. Only the type of the return object matters, the value will be ignored.
func (t *TopologyRest) ProducesObject(verb string) interface{} {
	return &mcmv1alpha1.ClusterStatus{}
}

var _ = rest.StorageMetadata(&TopologyRest{})

func (t *TopologyRest) New() runtime.Object {
	return &mcmv1alpha1.ClusterStatusTopology{}
}

var _ = rest.Creater(&TopologyRest{})

func (t *TopologyRest) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	clusterTopologyData, ok := obj.(*mcmv1alpha1.ClusterStatusTopology)
	if !ok {
		return nil, errors.NewBadRequest(fmt.Sprintf("not a cluster status topology: %#v", obj))
	}

	t.inserter.InsertData(clusterTopologyData)

	return &metav1.Status{
		Status:  metav1.StatusSuccess,
		Message: fmt.Sprintf("Successfully saved data in mongo"),
		Code:    http.StatusOK,
	}, nil
}
