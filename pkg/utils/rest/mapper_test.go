package rest

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"
)

func Test_Mapper_Methods(t *testing.T) {
	Convey("test Mapper MappingForGVK", t, func() {
		resources := []*restmapper.APIGroupResources{
			{
				Group: metav1.APIGroup{},
			},
		}
		mapper := NewFakeMapper(resources)
		gvk := schema.GroupVersionKind{
			Group:   "meta.k8s.io",
			Version: "v1",
			Kind:    "Pods",
		}
		_, err := mapper.MappingForGVK(gvk)
		So(err, ShouldBeError)
	})
	Convey("test Mapper Mapper", t, func() {
		resources := []*restmapper.APIGroupResources{
			{
				Group: metav1.APIGroup{},
			},
		}
		mapper := NewFakeMapper(resources)
		mapper.Mapper()
	})
	Convey("test Mapper MappingFor", t, func() {
		resources := []*restmapper.APIGroupResources{
			{
				Group: metav1.APIGroup{},
			},
		}
		mapper := NewFakeMapper(resources)
		_, err := mapper.MappingFor("pods")
		So(err, ShouldBeError)
	})
}
