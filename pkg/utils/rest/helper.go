package rest

import (
	"context"
	"encoding/base64"
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
	"k8s.io/klog"
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
	return req.Do(context.TODO()).Get()
}

type dynamicCodec struct{}

func (dynamicCodec) Decode(
	data []byte, gvk *schema.GroupVersionKind, obj runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(data, gvk, obj)
	if err != nil {
		return nil, nil, err
	}

	if _, ok := obj.(*metav1.Status); !ok && strings.EqualFold(gvk.Kind, "status") {
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
func (dynamicCodec) Identifier() runtime.Identifier {
	return ""
}

func ParseUserIdentity(idEncoded string) (id string) {
	if idEncoded == "" {
		return ""
	}
	idDecoded, err := base64.StdEncoding.DecodeString(idEncoded)
	if err != nil {
		klog.Error(err)
		return ""
	}
	userID := string(idDecoded)
	return userID
}

func ParseUserGroup(groupEncoded string) (groups []string) {
	if groupEncoded == "" {
		return nil
	}
	var userGroups []string
	userGroupDecoded, err := base64.StdEncoding.DecodeString(groupEncoded)
	if err != nil {
		klog.Error(err)
		return nil
	}
	userGroup := string(userGroupDecoded)

	groupArray := strings.Split(userGroup, ",")
	for i := 0; i < len(groupArray); i++ { //we accept system groups and icp groups
		if strings.HasPrefix(groupArray[i], "icp:") {
			groupElements := strings.Split(groupArray[i], ":")
			if len(groupElements) == 3 {
				newGroup := "mcm::" + groupElements[2] //convert icp group (which is team specific) to mcm group to do cluster level role binding
				var exist = false
				for j := 0; j < len(userGroups); j++ {
					if newGroup == userGroups[j] {
						exist = true
					}
				}
				if !exist {
					userGroups = append(userGroups, newGroup)
				}
			}
		} else if strings.HasPrefix(groupArray[i], "system:") {
			userGroups = append(userGroups, groupArray[i])
		}
	}
	return userGroups
}
