// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClusterPools implements ClusterPoolInterface
type FakeClusterPools struct {
	Fake *FakeHiveV1
	ns   string
}

var clusterpoolsResource = schema.GroupVersionResource{Group: "hive.openshift.io", Version: "v1", Resource: "clusterpools"}

var clusterpoolsKind = schema.GroupVersionKind{Group: "hive.openshift.io", Version: "v1", Kind: "ClusterPool"}

// Get takes name of the clusterPool, and returns the corresponding clusterPool object, and an error if there is any.
func (c *FakeClusterPools) Get(ctx context.Context, name string, options v1.GetOptions) (result *hivev1.ClusterPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(clusterpoolsResource, c.ns, name), &hivev1.ClusterPool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*hivev1.ClusterPool), err
}

// List takes label and field selectors, and returns the list of ClusterPools that match those selectors.
func (c *FakeClusterPools) List(ctx context.Context, opts v1.ListOptions) (result *hivev1.ClusterPoolList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(clusterpoolsResource, clusterpoolsKind, c.ns, opts), &hivev1.ClusterPoolList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &hivev1.ClusterPoolList{ListMeta: obj.(*hivev1.ClusterPoolList).ListMeta}
	for _, item := range obj.(*hivev1.ClusterPoolList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterPools.
func (c *FakeClusterPools) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(clusterpoolsResource, c.ns, opts))

}

// Create takes the representation of a clusterPool and creates it.  Returns the server's representation of the clusterPool, and an error, if there is any.
func (c *FakeClusterPools) Create(ctx context.Context, clusterPool *hivev1.ClusterPool, opts v1.CreateOptions) (result *hivev1.ClusterPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(clusterpoolsResource, c.ns, clusterPool), &hivev1.ClusterPool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*hivev1.ClusterPool), err
}

// Update takes the representation of a clusterPool and updates it. Returns the server's representation of the clusterPool, and an error, if there is any.
func (c *FakeClusterPools) Update(ctx context.Context, clusterPool *hivev1.ClusterPool, opts v1.UpdateOptions) (result *hivev1.ClusterPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(clusterpoolsResource, c.ns, clusterPool), &hivev1.ClusterPool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*hivev1.ClusterPool), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeClusterPools) UpdateStatus(ctx context.Context, clusterPool *hivev1.ClusterPool, opts v1.UpdateOptions) (*hivev1.ClusterPool, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(clusterpoolsResource, "status", c.ns, clusterPool), &hivev1.ClusterPool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*hivev1.ClusterPool), err
}

// Delete takes name of the clusterPool and deletes it. Returns an error if one occurs.
func (c *FakeClusterPools) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(clusterpoolsResource, c.ns, name, opts), &hivev1.ClusterPool{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterPools) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(clusterpoolsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &hivev1.ClusterPoolList{})
	return err
}

// Patch applies the patch and returns the patched clusterPool.
func (c *FakeClusterPools) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *hivev1.ClusterPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(clusterpoolsResource, c.ns, name, pt, data, subresources...), &hivev1.ClusterPool{})

	if obj == nil {
		return nil, err
	}
	return obj.(*hivev1.ClusterPool), err
}
