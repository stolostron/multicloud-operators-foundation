// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package useridentity

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/initializer"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/informers"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
)

const (
	// PluginName is name of admission plug-in
	PluginName = "HCMUserIdentity"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(io.Reader) (admission.Interface, error) {
		return NewUserIdentiyAnnotate()
	})
}

// annotateUserIdentity is an implementation of admission.Interface.
// If creating of updateing hcm api, set user identity and group annotation
type annotateUserIdentity struct {
	*admission.Handler
	lister rbaclisters.RoleBindingLister
}

var _ = initializer.WantsExternalKubeInformerFactory(&annotateUserIdentity{})

func MergeUserIdentityToAnnotations(
	userInfo authenticationv1.UserInfo,
	annotations map[string]string,
	namespace string,
	listers rbaclisters.RoleBindingLister,
) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	user := userInfo.Username

	filteredGroups := []string{}
	for _, group := range userInfo.Groups {
		groupArray := strings.Split(group, ":")

		// add group not created by iam
		if len(groupArray) != 3 {
			filteredGroups = append(filteredGroups, group)
			continue
		}

		// add group not from icp
		if groupArray[0] != "icp" {
			filteredGroups = append(filteredGroups, group)
			continue
		}

		// add iam default group
		if groupArray[1] == "default" {
			filteredGroups = append(filteredGroups, group)
			continue
		}

		_, err := listers.RoleBindings(namespace).Get(group)

		if err == nil {
			filteredGroups = append(filteredGroups, group)
		}
	}

	group := strings.Join(filteredGroups, ",")

	annotations[v1alpha1.UserIdentityAnnotation] = base64.StdEncoding.EncodeToString([]byte(user))
	annotations[v1alpha1.UserGroupAnnotation] = base64.StdEncoding.EncodeToString([]byte(group))
	return annotations
}

func transferToUserInfo(user user.Info) authenticationv1.UserInfo {
	return authenticationv1.UserInfo{
		Extra:    make(map[string]authenticationv1.ExtraValue),
		Groups:   user.GetGroups(),
		UID:      user.GetUID(),
		Username: user.GetName(),
	}
}

func (b *annotateUserIdentity) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	// we need to wait for our caches to warm
	if !b.WaitForReady() {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	obj := a.GetObject()
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil
	}
	annotations := accessor.GetAnnotations()
	userInfo := transferToUserInfo(a.GetUserInfo())
	resAnnotations := MergeUserIdentityToAnnotations(userInfo, annotations, a.GetNamespace(), b.lister)
	accessor.SetAnnotations(resAnnotations)

	return nil
}

// SetExternalKubeInformerFactory implements the WantsExternalKubeInformerFactory interface.
func (b *annotateUserIdentity) SetExternalKubeInformerFactory(f informers.SharedInformerFactory) {
	informer := f.Rbac().V1().RoleBindings()
	b.lister = informer.Lister()
	b.SetReadyFunc(informer.Informer().HasSynced)
}

func (b *annotateUserIdentity) ValidateInitialization() error {
	if b.lister == nil {
		return fmt.Errorf("missing client")
	}
	return nil
}

// NewUserIdentiyAnnotate creates a new admission control handler that
// annotate the object based on user identity
func NewUserIdentiyAnnotate() (admission.Interface, error) {
	return &annotateUserIdentity{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}, nil
}
