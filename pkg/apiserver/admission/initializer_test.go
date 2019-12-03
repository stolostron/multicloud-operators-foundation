// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package admission

import (
	"testing"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/internalclientset"
	informers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated/internalversion"
	"k8s.io/apiserver/pkg/admission"
)

type WantInternalHCMClientSet struct {
	cs internalclientset.Interface
}

func (receive *WantInternalHCMClientSet) Handles(o admission.Operation) bool { return false }
func (receive *WantInternalHCMClientSet) SetInternalHCMClientSet(cs internalclientset.Interface) {
	receive.cs = cs
}
func (receive *WantInternalHCMClientSet) ValidateInitialization() error { return nil }

var _ admission.Interface = &WantInternalHCMClientSet{}
var _ WantsInternalHCMClientSet = &WantInternalHCMClientSet{}

func TestWantHCMClientSet(t *testing.T) {
	target := NewPluginInitializer(nil, nil, nil, nil, nil, nil)
	WantInternalHCMClientSet := &WantInternalHCMClientSet{}
	target.Initialize(WantInternalHCMClientSet)
	if WantInternalHCMClientSet.cs != nil {
		t.Errorf("fake testing error")
	}
}

type WantInternalHCMInformerFactory struct {
	sf informers.SharedInformerFactory
}

func (receive *WantInternalHCMInformerFactory) Handles(o admission.Operation) bool { return false }
func (receive *WantInternalHCMInformerFactory) SetInternalHCMInformerFactory(sf informers.SharedInformerFactory) {
	receive.sf = sf
}
func (receive *WantInternalHCMInformerFactory) ValidateInitialization() error { return nil }

var _ admission.Interface = &WantInternalHCMInformerFactory{}
var _ WantsInternalHCMInformerFactory = &WantInternalHCMInformerFactory{}

func TestWantHCMInformerFactory(t *testing.T) {
	target := NewPluginInitializer(nil, nil, nil, nil, nil, nil)
	WantInternalHCMInformerFactory := &WantInternalHCMInformerFactory{}
	target.Initialize(WantInternalHCMInformerFactory)
	if WantInternalHCMInformerFactory.sf != nil {
		t.Errorf("fake testing error")
	}
}
