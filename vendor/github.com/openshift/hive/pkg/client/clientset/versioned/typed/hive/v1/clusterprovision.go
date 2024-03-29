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

// ClusterProvisionsGetter has a method to return a ClusterProvisionInterface.
// A group's client should implement this interface.
type ClusterProvisionsGetter interface {
	ClusterProvisions(namespace string) ClusterProvisionInterface
}

// ClusterProvisionInterface has methods to work with ClusterProvision resources.
type ClusterProvisionInterface interface {
	Create(ctx context.Context, clusterProvision *v1.ClusterProvision, opts metav1.CreateOptions) (*v1.ClusterProvision, error)
	Update(ctx context.Context, clusterProvision *v1.ClusterProvision, opts metav1.UpdateOptions) (*v1.ClusterProvision, error)
	UpdateStatus(ctx context.Context, clusterProvision *v1.ClusterProvision, opts metav1.UpdateOptions) (*v1.ClusterProvision, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ClusterProvision, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ClusterProvisionList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterProvision, err error)
	Apply(ctx context.Context, clusterProvision *hivev1.ClusterProvisionApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ClusterProvision, err error)
	ApplyStatus(ctx context.Context, clusterProvision *hivev1.ClusterProvisionApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ClusterProvision, err error)
	ClusterProvisionExpansion
}

// clusterProvisions implements ClusterProvisionInterface
type clusterProvisions struct {
	client rest.Interface
	ns     string
}

// newClusterProvisions returns a ClusterProvisions
func newClusterProvisions(c *HiveV1Client, namespace string) *clusterProvisions {
	return &clusterProvisions{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the clusterProvision, and returns the corresponding clusterProvision object, and an error if there is any.
func (c *clusterProvisions) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.ClusterProvision, err error) {
	result = &v1.ClusterProvision{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("clusterprovisions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterProvisions that match those selectors.
func (c *clusterProvisions) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ClusterProvisionList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ClusterProvisionList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("clusterprovisions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterProvisions.
func (c *clusterProvisions) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("clusterprovisions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a clusterProvision and creates it.  Returns the server's representation of the clusterProvision, and an error, if there is any.
func (c *clusterProvisions) Create(ctx context.Context, clusterProvision *v1.ClusterProvision, opts metav1.CreateOptions) (result *v1.ClusterProvision, err error) {
	result = &v1.ClusterProvision{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("clusterprovisions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterProvision).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a clusterProvision and updates it. Returns the server's representation of the clusterProvision, and an error, if there is any.
func (c *clusterProvisions) Update(ctx context.Context, clusterProvision *v1.ClusterProvision, opts metav1.UpdateOptions) (result *v1.ClusterProvision, err error) {
	result = &v1.ClusterProvision{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("clusterprovisions").
		Name(clusterProvision.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterProvision).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *clusterProvisions) UpdateStatus(ctx context.Context, clusterProvision *v1.ClusterProvision, opts metav1.UpdateOptions) (result *v1.ClusterProvision, err error) {
	result = &v1.ClusterProvision{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("clusterprovisions").
		Name(clusterProvision.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterProvision).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the clusterProvision and deletes it. Returns an error if one occurs.
func (c *clusterProvisions) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("clusterprovisions").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterProvisions) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("clusterprovisions").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched clusterProvision.
func (c *clusterProvisions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterProvision, err error) {
	result = &v1.ClusterProvision{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("clusterprovisions").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied clusterProvision.
func (c *clusterProvisions) Apply(ctx context.Context, clusterProvision *hivev1.ClusterProvisionApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ClusterProvision, err error) {
	if clusterProvision == nil {
		return nil, fmt.Errorf("clusterProvision provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(clusterProvision)
	if err != nil {
		return nil, err
	}
	name := clusterProvision.Name
	if name == nil {
		return nil, fmt.Errorf("clusterProvision.Name must be provided to Apply")
	}
	result = &v1.ClusterProvision{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("clusterprovisions").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *clusterProvisions) ApplyStatus(ctx context.Context, clusterProvision *hivev1.ClusterProvisionApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ClusterProvision, err error) {
	if clusterProvision == nil {
		return nil, fmt.Errorf("clusterProvision provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(clusterProvision)
	if err != nil {
		return nil, err
	}

	name := clusterProvision.Name
	if name == nil {
		return nil, fmt.Errorf("clusterProvision.Name must be provided to Apply")
	}

	result = &v1.ClusterProvision{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("clusterprovisions").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
