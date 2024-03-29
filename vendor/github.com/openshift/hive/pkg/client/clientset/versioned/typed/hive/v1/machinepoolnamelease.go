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

// MachinePoolNameLeasesGetter has a method to return a MachinePoolNameLeaseInterface.
// A group's client should implement this interface.
type MachinePoolNameLeasesGetter interface {
	MachinePoolNameLeases(namespace string) MachinePoolNameLeaseInterface
}

// MachinePoolNameLeaseInterface has methods to work with MachinePoolNameLease resources.
type MachinePoolNameLeaseInterface interface {
	Create(ctx context.Context, machinePoolNameLease *v1.MachinePoolNameLease, opts metav1.CreateOptions) (*v1.MachinePoolNameLease, error)
	Update(ctx context.Context, machinePoolNameLease *v1.MachinePoolNameLease, opts metav1.UpdateOptions) (*v1.MachinePoolNameLease, error)
	UpdateStatus(ctx context.Context, machinePoolNameLease *v1.MachinePoolNameLease, opts metav1.UpdateOptions) (*v1.MachinePoolNameLease, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.MachinePoolNameLease, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.MachinePoolNameLeaseList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.MachinePoolNameLease, err error)
	Apply(ctx context.Context, machinePoolNameLease *hivev1.MachinePoolNameLeaseApplyConfiguration, opts metav1.ApplyOptions) (result *v1.MachinePoolNameLease, err error)
	ApplyStatus(ctx context.Context, machinePoolNameLease *hivev1.MachinePoolNameLeaseApplyConfiguration, opts metav1.ApplyOptions) (result *v1.MachinePoolNameLease, err error)
	MachinePoolNameLeaseExpansion
}

// machinePoolNameLeases implements MachinePoolNameLeaseInterface
type machinePoolNameLeases struct {
	client rest.Interface
	ns     string
}

// newMachinePoolNameLeases returns a MachinePoolNameLeases
func newMachinePoolNameLeases(c *HiveV1Client, namespace string) *machinePoolNameLeases {
	return &machinePoolNameLeases{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the machinePoolNameLease, and returns the corresponding machinePoolNameLease object, and an error if there is any.
func (c *machinePoolNameLeases) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.MachinePoolNameLease, err error) {
	result = &v1.MachinePoolNameLease{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MachinePoolNameLeases that match those selectors.
func (c *machinePoolNameLeases) List(ctx context.Context, opts metav1.ListOptions) (result *v1.MachinePoolNameLeaseList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.MachinePoolNameLeaseList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested machinePoolNameLeases.
func (c *machinePoolNameLeases) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a machinePoolNameLease and creates it.  Returns the server's representation of the machinePoolNameLease, and an error, if there is any.
func (c *machinePoolNameLeases) Create(ctx context.Context, machinePoolNameLease *v1.MachinePoolNameLease, opts metav1.CreateOptions) (result *v1.MachinePoolNameLease, err error) {
	result = &v1.MachinePoolNameLease{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(machinePoolNameLease).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a machinePoolNameLease and updates it. Returns the server's representation of the machinePoolNameLease, and an error, if there is any.
func (c *machinePoolNameLeases) Update(ctx context.Context, machinePoolNameLease *v1.MachinePoolNameLease, opts metav1.UpdateOptions) (result *v1.MachinePoolNameLease, err error) {
	result = &v1.MachinePoolNameLease{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		Name(machinePoolNameLease.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(machinePoolNameLease).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *machinePoolNameLeases) UpdateStatus(ctx context.Context, machinePoolNameLease *v1.MachinePoolNameLease, opts metav1.UpdateOptions) (result *v1.MachinePoolNameLease, err error) {
	result = &v1.MachinePoolNameLease{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		Name(machinePoolNameLease.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(machinePoolNameLease).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the machinePoolNameLease and deletes it. Returns an error if one occurs.
func (c *machinePoolNameLeases) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *machinePoolNameLeases) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched machinePoolNameLease.
func (c *machinePoolNameLeases) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.MachinePoolNameLease, err error) {
	result = &v1.MachinePoolNameLease{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied machinePoolNameLease.
func (c *machinePoolNameLeases) Apply(ctx context.Context, machinePoolNameLease *hivev1.MachinePoolNameLeaseApplyConfiguration, opts metav1.ApplyOptions) (result *v1.MachinePoolNameLease, err error) {
	if machinePoolNameLease == nil {
		return nil, fmt.Errorf("machinePoolNameLease provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(machinePoolNameLease)
	if err != nil {
		return nil, err
	}
	name := machinePoolNameLease.Name
	if name == nil {
		return nil, fmt.Errorf("machinePoolNameLease.Name must be provided to Apply")
	}
	result = &v1.MachinePoolNameLease{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *machinePoolNameLeases) ApplyStatus(ctx context.Context, machinePoolNameLease *hivev1.MachinePoolNameLeaseApplyConfiguration, opts metav1.ApplyOptions) (result *v1.MachinePoolNameLease, err error) {
	if machinePoolNameLease == nil {
		return nil, fmt.Errorf("machinePoolNameLease provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(machinePoolNameLease)
	if err != nil {
		return nil, err
	}

	name := machinePoolNameLease.Name
	if name == nil {
		return nil, fmt.Errorf("machinePoolNameLease.Name must be provided to Apply")
	}

	result = &v1.MachinePoolNameLease{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("machinepoolnameleases").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
