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

// FakeMachinePools implements MachinePoolInterface
type FakeMachinePools struct {
	Fake *FakeHiveV1
	ns   string
}

var machinepoolsResource = v1.SchemeGroupVersion.WithResource("machinepools")

var machinepoolsKind = v1.SchemeGroupVersion.WithKind("MachinePool")

// Get takes name of the machinePool, and returns the corresponding machinePool object, and an error if there is any.
func (c *FakeMachinePools) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.MachinePool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(machinepoolsResource, c.ns, name), &v1.MachinePool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePool), err
}

// List takes label and field selectors, and returns the list of MachinePools that match those selectors.
func (c *FakeMachinePools) List(ctx context.Context, opts metav1.ListOptions) (result *v1.MachinePoolList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(machinepoolsResource, machinepoolsKind, c.ns, opts), &v1.MachinePoolList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.MachinePoolList{ListMeta: obj.(*v1.MachinePoolList).ListMeta}
	for _, item := range obj.(*v1.MachinePoolList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested machinePools.
func (c *FakeMachinePools) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(machinepoolsResource, c.ns, opts))

}

// Create takes the representation of a machinePool and creates it.  Returns the server's representation of the machinePool, and an error, if there is any.
func (c *FakeMachinePools) Create(ctx context.Context, machinePool *v1.MachinePool, opts metav1.CreateOptions) (result *v1.MachinePool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(machinepoolsResource, c.ns, machinePool), &v1.MachinePool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePool), err
}

// Update takes the representation of a machinePool and updates it. Returns the server's representation of the machinePool, and an error, if there is any.
func (c *FakeMachinePools) Update(ctx context.Context, machinePool *v1.MachinePool, opts metav1.UpdateOptions) (result *v1.MachinePool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(machinepoolsResource, c.ns, machinePool), &v1.MachinePool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePool), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMachinePools) UpdateStatus(ctx context.Context, machinePool *v1.MachinePool, opts metav1.UpdateOptions) (*v1.MachinePool, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(machinepoolsResource, "status", c.ns, machinePool), &v1.MachinePool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePool), err
}

// Delete takes name of the machinePool and deletes it. Returns an error if one occurs.
func (c *FakeMachinePools) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(machinepoolsResource, c.ns, name, opts), &v1.MachinePool{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMachinePools) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(machinepoolsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1.MachinePoolList{})
	return err
}

// Patch applies the patch and returns the patched machinePool.
func (c *FakeMachinePools) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.MachinePool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(machinepoolsResource, c.ns, name, pt, data, subresources...), &v1.MachinePool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePool), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied machinePool.
func (c *FakeMachinePools) Apply(ctx context.Context, machinePool *hivev1.MachinePoolApplyConfiguration, opts metav1.ApplyOptions) (result *v1.MachinePool, err error) {
	if machinePool == nil {
		return nil, fmt.Errorf("machinePool provided to Apply must not be nil")
	}
	data, err := json.Marshal(machinePool)
	if err != nil {
		return nil, err
	}
	name := machinePool.Name
	if name == nil {
		return nil, fmt.Errorf("machinePool.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(machinepoolsResource, c.ns, *name, types.ApplyPatchType, data), &v1.MachinePool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePool), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeMachinePools) ApplyStatus(ctx context.Context, machinePool *hivev1.MachinePoolApplyConfiguration, opts metav1.ApplyOptions) (result *v1.MachinePool, err error) {
	if machinePool == nil {
		return nil, fmt.Errorf("machinePool provided to Apply must not be nil")
	}
	data, err := json.Marshal(machinePool)
	if err != nil {
		return nil, err
	}
	name := machinePool.Name
	if name == nil {
		return nil, fmt.Errorf("machinePool.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(machinepoolsResource, c.ns, *name, types.ApplyPatchType, data, "status"), &v1.MachinePool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.MachinePool), err
}
