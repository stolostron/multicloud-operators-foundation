// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1 "github.com/openshift/hive/apis/hive/v1"
	hivev1 "github.com/openshift/hive/pkg/client/applyconfiguration/hive/v1"
	scheme "github.com/openshift/hive/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClusterRelocatesGetter has a method to return a ClusterRelocateInterface.
// A group's client should implement this interface.
type ClusterRelocatesGetter interface {
	ClusterRelocates() ClusterRelocateInterface
}

// ClusterRelocateInterface has methods to work with ClusterRelocate resources.
type ClusterRelocateInterface interface {
	Create(ctx context.Context, clusterRelocate *v1.ClusterRelocate, opts metav1.CreateOptions) (*v1.ClusterRelocate, error)
	Update(ctx context.Context, clusterRelocate *v1.ClusterRelocate, opts metav1.UpdateOptions) (*v1.ClusterRelocate, error)
	UpdateStatus(ctx context.Context, clusterRelocate *v1.ClusterRelocate, opts metav1.UpdateOptions) (*v1.ClusterRelocate, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ClusterRelocate, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ClusterRelocateList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterRelocate, err error)
	Apply(ctx context.Context, clusterRelocate *hivev1.ClusterRelocateApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ClusterRelocate, err error)
	ApplyStatus(ctx context.Context, clusterRelocate *hivev1.ClusterRelocateApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ClusterRelocate, err error)
	ClusterRelocateExpansion
}

// clusterRelocates implements ClusterRelocateInterface
type clusterRelocates struct {
	client rest.Interface
}

// newClusterRelocates returns a ClusterRelocates
func newClusterRelocates(c *HiveV1Client) *clusterRelocates {
	return &clusterRelocates{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterRelocate, and returns the corresponding clusterRelocate object, and an error if there is any.
func (c *clusterRelocates) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.ClusterRelocate, err error) {
	result = &v1.ClusterRelocate{}
	err = c.client.Get().
		Resource("clusterrelocates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterRelocates that match those selectors.
func (c *clusterRelocates) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ClusterRelocateList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ClusterRelocateList{}
	err = c.client.Get().
		Resource("clusterrelocates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterRelocates.
func (c *clusterRelocates) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("clusterrelocates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a clusterRelocate and creates it.  Returns the server's representation of the clusterRelocate, and an error, if there is any.
func (c *clusterRelocates) Create(ctx context.Context, clusterRelocate *v1.ClusterRelocate, opts metav1.CreateOptions) (result *v1.ClusterRelocate, err error) {
	result = &v1.ClusterRelocate{}
	err = c.client.Post().
		Resource("clusterrelocates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterRelocate).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a clusterRelocate and updates it. Returns the server's representation of the clusterRelocate, and an error, if there is any.
func (c *clusterRelocates) Update(ctx context.Context, clusterRelocate *v1.ClusterRelocate, opts metav1.UpdateOptions) (result *v1.ClusterRelocate, err error) {
	result = &v1.ClusterRelocate{}
	err = c.client.Put().
		Resource("clusterrelocates").
		Name(clusterRelocate.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterRelocate).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *clusterRelocates) UpdateStatus(ctx context.Context, clusterRelocate *v1.ClusterRelocate, opts metav1.UpdateOptions) (result *v1.ClusterRelocate, err error) {
	result = &v1.ClusterRelocate{}
	err = c.client.Put().
		Resource("clusterrelocates").
		Name(clusterRelocate.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterRelocate).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the clusterRelocate and deletes it. Returns an error if one occurs.
func (c *clusterRelocates) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusterrelocates").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterRelocates) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("clusterrelocates").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched clusterRelocate.
func (c *clusterRelocates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterRelocate, err error) {
	result = &v1.ClusterRelocate{}
	err = c.client.Patch(pt).
		Resource("clusterrelocates").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied clusterRelocate.
func (c *clusterRelocates) Apply(ctx context.Context, clusterRelocate *hivev1.ClusterRelocateApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ClusterRelocate, err error) {
	if clusterRelocate == nil {
		return nil, fmt.Errorf("clusterRelocate provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(clusterRelocate)
	if err != nil {
		return nil, err
	}
	name := clusterRelocate.Name
	if name == nil {
		return nil, fmt.Errorf("clusterRelocate.Name must be provided to Apply")
	}
	result = &v1.ClusterRelocate{}
	err = c.client.Patch(types.ApplyPatchType).
		Resource("clusterrelocates").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *clusterRelocates) ApplyStatus(ctx context.Context, clusterRelocate *hivev1.ClusterRelocateApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ClusterRelocate, err error) {
	if clusterRelocate == nil {
		return nil, fmt.Errorf("clusterRelocate provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(clusterRelocate)
	if err != nil {
		return nil, err
	}

	name := clusterRelocate.Name
	if name == nil {
		return nil, fmt.Errorf("clusterRelocate.Name must be provided to Apply")
	}

	result = &v1.ClusterRelocate{}
	err = c.client.Patch(types.ApplyPatchType).
		Resource("clusterrelocates").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
