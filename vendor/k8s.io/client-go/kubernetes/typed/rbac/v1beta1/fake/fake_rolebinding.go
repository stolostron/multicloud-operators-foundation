/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1beta1 "k8s.io/api/rbac/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rbacv1beta1 "k8s.io/client-go/applyconfigurations/rbac/v1beta1"
	testing "k8s.io/client-go/testing"
)

// FakeRoleBindings implements RoleBindingInterface
type FakeRoleBindings struct {
	Fake *FakeRbacV1beta1
	ns   string
}

var rolebindingsResource = v1beta1.SchemeGroupVersion.WithResource("rolebindings")

var rolebindingsKind = v1beta1.SchemeGroupVersion.WithKind("RoleBinding")

// Get takes name of the roleBinding, and returns the corresponding roleBinding object, and an error if there is any.
func (c *FakeRoleBindings) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.RoleBinding, err error) {
	emptyResult := &v1beta1.RoleBinding{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(rolebindingsResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.RoleBinding), err
}

// List takes label and field selectors, and returns the list of RoleBindings that match those selectors.
func (c *FakeRoleBindings) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.RoleBindingList, err error) {
	emptyResult := &v1beta1.RoleBindingList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(rolebindingsResource, rolebindingsKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.RoleBindingList{ListMeta: obj.(*v1beta1.RoleBindingList).ListMeta}
	for _, item := range obj.(*v1beta1.RoleBindingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested roleBindings.
func (c *FakeRoleBindings) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(rolebindingsResource, c.ns, opts))

}

// Create takes the representation of a roleBinding and creates it.  Returns the server's representation of the roleBinding, and an error, if there is any.
func (c *FakeRoleBindings) Create(ctx context.Context, roleBinding *v1beta1.RoleBinding, opts v1.CreateOptions) (result *v1beta1.RoleBinding, err error) {
	emptyResult := &v1beta1.RoleBinding{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(rolebindingsResource, c.ns, roleBinding, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.RoleBinding), err
}

// Update takes the representation of a roleBinding and updates it. Returns the server's representation of the roleBinding, and an error, if there is any.
func (c *FakeRoleBindings) Update(ctx context.Context, roleBinding *v1beta1.RoleBinding, opts v1.UpdateOptions) (result *v1beta1.RoleBinding, err error) {
	emptyResult := &v1beta1.RoleBinding{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(rolebindingsResource, c.ns, roleBinding, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.RoleBinding), err
}

// Delete takes name of the roleBinding and deletes it. Returns an error if one occurs.
func (c *FakeRoleBindings) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(rolebindingsResource, c.ns, name, opts), &v1beta1.RoleBinding{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRoleBindings) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(rolebindingsResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.RoleBindingList{})
	return err
}

// Patch applies the patch and returns the patched roleBinding.
func (c *FakeRoleBindings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.RoleBinding, err error) {
	emptyResult := &v1beta1.RoleBinding{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(rolebindingsResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.RoleBinding), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied roleBinding.
func (c *FakeRoleBindings) Apply(ctx context.Context, roleBinding *rbacv1beta1.RoleBindingApplyConfiguration, opts v1.ApplyOptions) (result *v1beta1.RoleBinding, err error) {
	if roleBinding == nil {
		return nil, fmt.Errorf("roleBinding provided to Apply must not be nil")
	}
	data, err := json.Marshal(roleBinding)
	if err != nil {
		return nil, err
	}
	name := roleBinding.Name
	if name == nil {
		return nil, fmt.Errorf("roleBinding.Name must be provided to Apply")
	}
	emptyResult := &v1beta1.RoleBinding{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(rolebindingsResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1beta1.RoleBinding), err
}
