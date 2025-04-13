// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "github.com/openshift/api/config/v1"
	configv1 "github.com/openshift/client-go/config/applyconfigurations/config/v1"
	typedconfigv1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	gentype "k8s.io/client-go/gentype"
)

// fakeImageDigestMirrorSets implements ImageDigestMirrorSetInterface
type fakeImageDigestMirrorSets struct {
	*gentype.FakeClientWithListAndApply[*v1.ImageDigestMirrorSet, *v1.ImageDigestMirrorSetList, *configv1.ImageDigestMirrorSetApplyConfiguration]
	Fake *FakeConfigV1
}

func newFakeImageDigestMirrorSets(fake *FakeConfigV1) typedconfigv1.ImageDigestMirrorSetInterface {
	return &fakeImageDigestMirrorSets{
		gentype.NewFakeClientWithListAndApply[*v1.ImageDigestMirrorSet, *v1.ImageDigestMirrorSetList, *configv1.ImageDigestMirrorSetApplyConfiguration](
			fake.Fake,
			"",
			v1.SchemeGroupVersion.WithResource("imagedigestmirrorsets"),
			v1.SchemeGroupVersion.WithKind("ImageDigestMirrorSet"),
			func() *v1.ImageDigestMirrorSet { return &v1.ImageDigestMirrorSet{} },
			func() *v1.ImageDigestMirrorSetList { return &v1.ImageDigestMirrorSetList{} },
			func(dst, src *v1.ImageDigestMirrorSetList) { dst.ListMeta = src.ListMeta },
			func(list *v1.ImageDigestMirrorSetList) []*v1.ImageDigestMirrorSet {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1.ImageDigestMirrorSetList, items []*v1.ImageDigestMirrorSet) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
