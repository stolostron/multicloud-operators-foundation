// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package useridentity

import (
	"encoding/base64"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/admission"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	hadmission "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apiserver/admission"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/internalclientset"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/internalclientset/fake"
	informers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated/internalversion"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

// newHandlerForTest returns a configured handler for testing.
func newHandlerForTest(
	internalClient internalclientset.Interface,
	kubeclient kubeclientset.Interface) (admission.Interface, kubeinformers.SharedInformerFactory, error) {
	f := informers.NewSharedInformerFactory(internalClient, 5*time.Minute)
	handler, err := NewUserIdentiyAnnotate()
	if err != nil {
		return nil, nil, err
	}
	kf := kubeinformers.NewSharedInformerFactory(kubeclient, 5*time.Minute)
	pluginInitializer := hadmission.NewPluginInitializer(internalClient, f, kubeclient, kf, nil, nil)
	pluginInitializer.Initialize(handler)
	err = admission.ValidateInitialization(handler)
	return handler, kf, err
}

// newFakeHCMClientForTest creates a fake clientset that returns a
// worklist with the given work as the single list item.
func newFakeHCMClientForTest() *fake.Clientset {
	return &fake.Clientset{}
}

// newWork returns a new work for the specified namespace.
func newWork(namespace string) mcm.Work {
	work := mcm.Work{
		ObjectMeta: metav1.ObjectMeta{Name: "instance", Namespace: namespace},
		Spec:       mcm.WorkSpec{},
	}
	return work
}

func TestUserIdentityAnnotate(t *testing.T) {
	fakeClient := newFakeHCMClientForTest()
	fakeKubeClient := kubefake.NewSimpleClientset()
	handler, informerFactory, err := newHandlerForTest(fakeClient, fakeKubeClient)
	if err != nil {
		t.Errorf("unexpected error initializing handler: %v", err)
	}
	informerFactory.Start(wait.NeverStop)
	work := newWork("dummy")
	userInfo := &user.DefaultInfo{
		Name:   "user1",
		Groups: []string{"group1", "group2"},
	}
	err = handler.(admission.MutationInterface).Admit(
		admission.NewAttributesRecord(
			&work,
			nil,
			mcm.Kind("Work").WithVersion("version"),
			work.Namespace,
			work.Name,
			mcm.Resource("works").WithVersion("version"), "", admission.Create, true, userInfo))
	if err != nil {
		t.Errorf("unexpected error %q returned from admission handler.", err.Error())
	}

	annotations := work.GetAnnotations()
	user, ok := annotations[v1alpha1.UserIdentityAnnotation]
	if !ok {
		t.Errorf("User is not found")
	}
	decodedUser, err := base64.StdEncoding.DecodeString(user)
	if err != nil {
		t.Errorf("Failed to decode user identity: %q", err.Error())
	}

	if string(decodedUser) != "user1" {
		t.Errorf("User identity does not match, expected %s, actual %s", "user1", string(decodedUser))
	}

	groups, ok := annotations[v1alpha1.UserGroupAnnotation]
	if !ok {
		t.Errorf("Group is not found")
	}
	decodedGroup, err := base64.StdEncoding.DecodeString(groups)
	if err != nil {
		t.Errorf("Failed to decode group: %q", err.Error())
	}

	if string(decodedGroup) != "group1,group2" {
		t.Errorf("User identity does not match, expected %s, actual %s", "group1,group2", string(decodedGroup))
	}
}

func TestUserIdentityAnnotateWithIAM(t *testing.T) {
	rolebinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "icp:dev:admin",
			Namespace: "dummy",
		},
	}
	fakeClient := newFakeHCMClientForTest()
	fakeKubeClient := kubefake.NewSimpleClientset(rolebinding)
	handler, informerFactory, err := newHandlerForTest(fakeClient, fakeKubeClient)
	if err != nil {
		t.Errorf("unexpected error initializing handler: %v", err)
	}
	informerFactory.Start(wait.NeverStop)
	work := newWork("dummy")
	userInfo := &user.DefaultInfo{
		Name:   "user1",
		Groups: []string{"icp:dev:admin", "icp:test:admin", "icp:default:member", "system:authenticated"},
	}
	err = handler.(admission.MutationInterface).Admit(
		admission.NewAttributesRecord(
			&work,
			nil,
			mcm.Kind("Work").WithVersion("version"),
			work.Namespace,
			work.Name,
			mcm.Resource("works").WithVersion("version"), "", admission.Create, true, userInfo))
	if err != nil {
		t.Errorf("unexpected error %q returned from admission handler.", err.Error())
	}

	annotations := work.GetAnnotations()
	user, ok := annotations[v1alpha1.UserIdentityAnnotation]
	if !ok {
		t.Errorf("User is not found")
	}
	decodedUser, err := base64.StdEncoding.DecodeString(user)
	if err != nil {
		t.Errorf("Failed to decode user identity: %q", err.Error())
	}

	if string(decodedUser) != "user1" {
		t.Errorf("User identity does not match, expected %s, actual %s", "user1", string(decodedUser))
	}

	groups, ok := annotations[v1alpha1.UserGroupAnnotation]
	if !ok {
		t.Errorf("Group is not found")
	}
	decodedGroup, err := base64.StdEncoding.DecodeString(groups)
	if err != nil {
		t.Errorf("Failed to decode group: %q", err.Error())
	}

	if string(decodedGroup) != "icp:dev:admin,icp:default:member,system:authenticated" {
		t.Errorf("User identity does not match, expected %s, actual %s", "group1,group2", string(decodedGroup))
	}
}
