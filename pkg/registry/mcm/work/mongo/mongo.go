// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mongo

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	mcmstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
)

type WorkResultRest struct {
	store storage.Interface
}

func NewWorkResultRest(optsGetter generic.RESTOptionsGetter, storageOptions *mcmstorage.StorageOptions) (*WorkResultRest, error) {
	store, err := mcmstorage.NewMCMStorage(storageOptions, mcm.Kind("ResourceViewResult"))
	if err != nil {
		return nil, err
	}
	return &WorkResultRest{store: store}, nil
}

// ProducesMIMETypes returns a list of the MIME types the specified HTTP verb (GET, POST, DELETE,
// PATCH) can respond with.
func (t *WorkResultRest) ProducesMIMETypes(verb string) []string {
	return []string{"application/json"}
}

// ProducesObject returns an object the specified HTTP verb respond with. It will overwrite storage object if
// it is not nil. Only the type of the return object matters, the value will be ignored.
func (t *WorkResultRest) ProducesObject(verb string) interface{} {
	return &v1alpha1.ResourceViewResult{}
}

var _ = rest.StorageMetadata(&WorkResultRest{})

func (t *WorkResultRest) New() runtime.Object {
	return &mcm.ResourceViewResult{}
}

var _ = rest.Creater(&WorkResultRest{})

func (t *WorkResultRest) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	result, ok := obj.(*mcm.ResourceViewResult)
	if !ok {
		return nil, errors.NewBadRequest(fmt.Sprintf("not a work result: %#v", obj))
	}
	key, _ := genericregistry.NamespaceKeyFunc(ctx, "", result.Name)
	key = strings.TrimPrefix(key, "/")

	returnResult := &mcm.ResourceViewResult{}
	err := t.store.GuaranteedUpdate(ctx, key, returnResult, true, &storage.Preconditions{}, func(existing runtime.Object, res storage.ResponseMeta) (runtime.Object, *uint64, error) {
		return result, nil, nil
	})

	if err != nil {
		return nil, err
	}

	return &metav1.Status{
		Status:  metav1.StatusSuccess,
		Message: fmt.Sprintf("Successfully saved data in mongo"),
		Code:    http.StatusOK,
	}, nil
}
