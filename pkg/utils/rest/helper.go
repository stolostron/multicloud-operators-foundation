// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// IBM Confidential
// OCO Source Materials
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.

package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// Helper provides methods for retrieving or mutating a RESTful
// resource.
type Helper struct {
	// The name of this resource as the server would recognize it
	Resource string
	// A RESTClient capable of mutating this resource.
	RESTClient *rest.RESTClient
	// True if the resource type is scoped to namespaces
	NamespaceScoped bool
	// Enable server print
	ServerPrint bool
}

// NewHelper creates a Helper from a ResourceMapping
func NewHelper(config *rest.Config, mapping *meta.RESTMapping, serverPrint bool) (*Helper, error) {
	configShallowCopy := *config

	gv := mapping.GroupVersionKind.GroupVersion()
	configShallowCopy.GroupVersion = &gv
	if len(gv.Group) == 0 {
		configShallowCopy.APIPath = "/api"
	} else {
		configShallowCopy.APIPath = "/apis"
	}

	var jsonInfo runtime.SerializerInfo
	for _, info := range scheme.Codecs.SupportedMediaTypes() {
		if info.MediaType == runtime.ContentTypeJSON {
			jsonInfo = info
			break
		}
	}
	jsonInfo.Serializer = dynamicCodec{}
	jsonInfo.PrettySerializer = nil
	configShallowCopy.NegotiatedSerializer = serializer.NegotiatedSerializerWrapper(jsonInfo)

	restClient, err := rest.RESTClientFor(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	return &Helper{
		Resource:        mapping.Resource.Resource,
		RESTClient:      restClient,
		NamespaceScoped: mapping.Scope.Name() == meta.RESTScopeNameNamespace,
		ServerPrint:     serverPrint,
	}, nil
}

// List return object list
func (m *Helper) List(namespace string, options *metav1.ListOptions) (runtime.Object, error) {
	group := metav1beta1.GroupName
	version := metav1beta1.SchemeGroupVersion.Version

	req := m.RESTClient.Get().
		NamespaceIfScoped(namespace, m.NamespaceScoped).
		Resource(m.Resource).
		VersionedParams(options, metav1.ParameterCodec)

	if m.ServerPrint {
		tableParam := fmt.Sprintf("application/json;as=Table;v=%s;g=%s, application/json", version, group)
		req.SetHeader("Accept", tableParam)
	}
	return req.Do().Get()
}

type dynamicCodec struct{}

func (dynamicCodec) Decode(data []byte, gvk *schema.GroupVersionKind, obj runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(data, gvk, obj)
	if err != nil {
		return nil, nil, err
	}

	if _, ok := obj.(*metav1.Status); !ok && strings.ToLower(gvk.Kind) == "status" {
		obj = &metav1.Status{}
		err := json.Unmarshal(data, obj)
		if err != nil {
			return nil, nil, err
		}
	}

	return obj, gvk, nil
}

func (dynamicCodec) Encode(obj runtime.Object, w io.Writer) error {
	return unstructured.UnstructuredJSONScheme.Encode(obj, w)
}
