// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterletca

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	hcmadmission "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apiserver/admission"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/internalclientset"
	informers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated/internalversion"
	listers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/listers_generated/mcm/internalversion"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

const (
	// PluginName is name of admission plug-in
	PluginName = "KlusterletCA"
)

var _ admission.MutationInterface = &klusterletCAAppend{}
var _ = hcmadmission.WantsInternalHCMClientSet(&klusterletCAAppend{})
var _ = hcmadmission.WantsInternalHCMInformerFactory(&klusterletCAAppend{})

// Register registers a plugin
func Register(plugins *admission.Plugins, caFile *string) {
	plugins.Register(PluginName, func(io.Reader) (admission.Interface, error) {
		return NewKlusterletCAAppend(caFile)
	})
}

// annotateUserIdentity is an implementation of admission.Interface.
// If creating of updateing hcm api, set user identity and group annotation
type klusterletCAAppend struct {
	caData []byte
	client internalclientset.Interface
	lister listers.ClusterStatusLister
	*admission.Handler
}

func (b *klusterletCAAppend) Admit(a admission.Attributes) error {
	if shouldIgnore(a) {
		return nil
	}

	// we need to wait for our caches to warm
	if !b.WaitForReady() {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	clusterStatus, err := b.lister.ClusterStatuses(a.GetNamespace()).Get(a.GetName())
	if err != nil {
		return nil
	}

	if reflect.DeepEqual(b.caData, clusterStatus.Spec.KlusterletCA) {
		return nil
	}

	clusterStatus.Spec.KlusterletCA = b.caData
	b.client.Mcm().ClusterStatuses(a.GetNamespace()).Update(clusterStatus)
	return nil
}

func (b *klusterletCAAppend) SetInternalHCMClientSet(client internalclientset.Interface) {
	b.client = client
}

func (b *klusterletCAAppend) SetInternalHCMInformerFactory(f informers.SharedInformerFactory) {
	informer := f.Mcm().InternalVersion().ClusterStatuses()
	b.lister = informer.Lister()
	b.SetReadyFunc(informer.Informer().HasSynced)
}

func (b *klusterletCAAppend) ValidateInitialization() error {
	if b.client == nil {
		return fmt.Errorf("missing client")
	}
	return nil
}

// NewKlusterletCAAppend creates a new admission control handler that
// add ca data to clusterstastus object
func NewKlusterletCAAppend(caFile *string) (admission.Interface, error) {
	pemBlock, err := ioutil.ReadFile(*caFile)
	if err != nil {
		return nil, err
	}

	return &klusterletCAAppend{
		caData:  pemBlock,
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}, nil
}

func shouldIgnore(a admission.Attributes) bool {
	if a.GetResource().GroupResource() != v1alpha1.Resource("clusters") {
		return true
	}
	obj := a.GetObject()
	if obj == nil {
		return true
	}
	_, ok := obj.(*v1alpha1.Cluster)
	return !ok
}
