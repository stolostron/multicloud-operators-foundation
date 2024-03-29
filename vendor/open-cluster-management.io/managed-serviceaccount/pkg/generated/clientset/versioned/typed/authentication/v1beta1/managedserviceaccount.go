/*
Copyright 2021.

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

package v1beta1

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1beta1 "open-cluster-management.io/managed-serviceaccount/apis/authentication/v1beta1"
	scheme "open-cluster-management.io/managed-serviceaccount/pkg/generated/clientset/versioned/scheme"
)

// ManagedServiceAccountsGetter has a method to return a ManagedServiceAccountInterface.
// A group's client should implement this interface.
type ManagedServiceAccountsGetter interface {
	ManagedServiceAccounts(namespace string) ManagedServiceAccountInterface
}

// ManagedServiceAccountInterface has methods to work with ManagedServiceAccount resources.
type ManagedServiceAccountInterface interface {
	Create(ctx context.Context, managedServiceAccount *v1beta1.ManagedServiceAccount, opts v1.CreateOptions) (*v1beta1.ManagedServiceAccount, error)
	Update(ctx context.Context, managedServiceAccount *v1beta1.ManagedServiceAccount, opts v1.UpdateOptions) (*v1beta1.ManagedServiceAccount, error)
	UpdateStatus(ctx context.Context, managedServiceAccount *v1beta1.ManagedServiceAccount, opts v1.UpdateOptions) (*v1beta1.ManagedServiceAccount, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta1.ManagedServiceAccount, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta1.ManagedServiceAccountList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.ManagedServiceAccount, err error)
	ManagedServiceAccountExpansion
}

// managedServiceAccounts implements ManagedServiceAccountInterface
type managedServiceAccounts struct {
	client rest.Interface
	ns     string
}

// newManagedServiceAccounts returns a ManagedServiceAccounts
func newManagedServiceAccounts(c *AuthenticationV1beta1Client, namespace string) *managedServiceAccounts {
	return &managedServiceAccounts{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the managedServiceAccount, and returns the corresponding managedServiceAccount object, and an error if there is any.
func (c *managedServiceAccounts) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.ManagedServiceAccount, err error) {
	result = &v1beta1.ManagedServiceAccount{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("managedserviceaccounts").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ManagedServiceAccounts that match those selectors.
func (c *managedServiceAccounts) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.ManagedServiceAccountList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta1.ManagedServiceAccountList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("managedserviceaccounts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested managedServiceAccounts.
func (c *managedServiceAccounts) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("managedserviceaccounts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a managedServiceAccount and creates it.  Returns the server's representation of the managedServiceAccount, and an error, if there is any.
func (c *managedServiceAccounts) Create(ctx context.Context, managedServiceAccount *v1beta1.ManagedServiceAccount, opts v1.CreateOptions) (result *v1beta1.ManagedServiceAccount, err error) {
	result = &v1beta1.ManagedServiceAccount{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("managedserviceaccounts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(managedServiceAccount).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a managedServiceAccount and updates it. Returns the server's representation of the managedServiceAccount, and an error, if there is any.
func (c *managedServiceAccounts) Update(ctx context.Context, managedServiceAccount *v1beta1.ManagedServiceAccount, opts v1.UpdateOptions) (result *v1beta1.ManagedServiceAccount, err error) {
	result = &v1beta1.ManagedServiceAccount{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("managedserviceaccounts").
		Name(managedServiceAccount.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(managedServiceAccount).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *managedServiceAccounts) UpdateStatus(ctx context.Context, managedServiceAccount *v1beta1.ManagedServiceAccount, opts v1.UpdateOptions) (result *v1beta1.ManagedServiceAccount, err error) {
	result = &v1beta1.ManagedServiceAccount{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("managedserviceaccounts").
		Name(managedServiceAccount.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(managedServiceAccount).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the managedServiceAccount and deletes it. Returns an error if one occurs.
func (c *managedServiceAccounts) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("managedserviceaccounts").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *managedServiceAccounts) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("managedserviceaccounts").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched managedServiceAccount.
func (c *managedServiceAccounts) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.ManagedServiceAccount, err error) {
	result = &v1beta1.ManagedServiceAccount{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("managedserviceaccounts").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
