// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1 "github.com/openshift/hive/apis/hive/v1"
	hivev1 "github.com/openshift/hive/pkg/client/applyconfiguration/hive/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMachinePoolNameLeases implements MachinePoolNameLeaseInterface
type FakeMachinePoolNameLeases struct {
	Fake *FakeHiveV1
	ns   string
}

var machinepoolnameleasesResource = v1.SchemeGroupVersion.WithResource("machinepoolnameleases")

var machinepoolnameleasesKind = v1.SchemeGroupVersion.WithKind("MachinePoolNameLease")

// Get takes name of the machinePoolNameLease, and returns the corresponding machinePoolNameLease object, and an error if there is any.
func (c *FakeMachinePoolNameLeases) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.MachinePoolNameLease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(machinepoolnameleasesResource, c.ns, name), &v1.MachinePoolNameLease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePoolNameLease), err
}

// List takes label and field selectors, and returns the list of MachinePoolNameLeases that match those selectors.
func (c *FakeMachinePoolNameLeases) List(ctx context.Context, opts metav1.ListOptions) (result *v1.MachinePoolNameLeaseList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(machinepoolnameleasesResource, machinepoolnameleasesKind, c.ns, opts), &v1.MachinePoolNameLeaseList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.MachinePoolNameLeaseList{ListMeta: obj.(*v1.MachinePoolNameLeaseList).ListMeta}
	for _, item := range obj.(*v1.MachinePoolNameLeaseList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested machinePoolNameLeases.
func (c *FakeMachinePoolNameLeases) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(machinepoolnameleasesResource, c.ns, opts))

}

// Create takes the representation of a machinePoolNameLease and creates it.  Returns the server's representation of the machinePoolNameLease, and an error, if there is any.
func (c *FakeMachinePoolNameLeases) Create(ctx context.Context, machinePoolNameLease *v1.MachinePoolNameLease, opts metav1.CreateOptions) (result *v1.MachinePoolNameLease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(machinepoolnameleasesResource, c.ns, machinePoolNameLease), &v1.MachinePoolNameLease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePoolNameLease), err
}

// Update takes the representation of a machinePoolNameLease and updates it. Returns the server's representation of the machinePoolNameLease, and an error, if there is any.
func (c *FakeMachinePoolNameLeases) Update(ctx context.Context, machinePoolNameLease *v1.MachinePoolNameLease, opts metav1.UpdateOptions) (result *v1.MachinePoolNameLease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(machinepoolnameleasesResource, c.ns, machinePoolNameLease), &v1.MachinePoolNameLease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePoolNameLease), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMachinePoolNameLeases) UpdateStatus(ctx context.Context, machinePoolNameLease *v1.MachinePoolNameLease, opts metav1.UpdateOptions) (*v1.MachinePoolNameLease, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(machinepoolnameleasesResource, "status", c.ns, machinePoolNameLease), &v1.MachinePoolNameLease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePoolNameLease), err
}

// Delete takes name of the machinePoolNameLease and deletes it. Returns an error if one occurs.
func (c *FakeMachinePoolNameLeases) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(machinepoolnameleasesResource, c.ns, name, opts), &v1.MachinePoolNameLease{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMachinePoolNameLeases) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(machinepoolnameleasesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1.MachinePoolNameLeaseList{})
	return err
}

// Patch applies the patch and returns the patched machinePoolNameLease.
func (c *FakeMachinePoolNameLeases) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.MachinePoolNameLease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(machinepoolnameleasesResource, c.ns, name, pt, data, subresources...), &v1.MachinePoolNameLease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePoolNameLease), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied machinePoolNameLease.
func (c *FakeMachinePoolNameLeases) Apply(ctx context.Context, machinePoolNameLease *hivev1.MachinePoolNameLeaseApplyConfiguration, opts metav1.ApplyOptions) (result *v1.MachinePoolNameLease, err error) {
	if machinePoolNameLease == nil {
		return nil, fmt.Errorf("machinePoolNameLease provided to Apply must not be nil")
	}
	data, err := json.Marshal(machinePoolNameLease)
	if err != nil {
		return nil, err
	}
	name := machinePoolNameLease.Name
	if name == nil {
		return nil, fmt.Errorf("machinePoolNameLease.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(machinepoolnameleasesResource, c.ns, *name, types.ApplyPatchType, data), &v1.MachinePoolNameLease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePoolNameLease), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeMachinePoolNameLeases) ApplyStatus(ctx context.Context, machinePoolNameLease *hivev1.MachinePoolNameLeaseApplyConfiguration, opts metav1.ApplyOptions) (result *v1.MachinePoolNameLease, err error) {
	if machinePoolNameLease == nil {
		return nil, fmt.Errorf("machinePoolNameLease provided to Apply must not be nil")
	}
	data, err := json.Marshal(machinePoolNameLease)
	if err != nil {
		return nil, err
	}
	name := machinePoolNameLease.Name
	if name == nil {
		return nil, fmt.Errorf("machinePoolNameLease.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(machinepoolnameleasesResource, c.ns, *name, types.ApplyPatchType, data, "status"), &v1.MachinePoolNameLease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePoolNameLease), err
}
