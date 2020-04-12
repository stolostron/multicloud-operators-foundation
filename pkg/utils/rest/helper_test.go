package rest

import (
	"encoding/base64"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type fakeRestScope struct {
	name meta.RESTScopeName
}

func (r *fakeRestScope) Name() meta.RESTScopeName {
	return r.name
}

func Test_Helper_Methods(t *testing.T) {
	Convey("test Helper Methods", t, func() {
		mapper := &meta.RESTMapping{
			GroupVersionKind: schema.GroupVersionKind{Kind: "Foo", Version: "v1alpha1"},
			Scope:            &fakeRestScope{name: "foo"},
		}
		helper, err := NewHelper(&rest.Config{}, mapper, false)
		So(err, ShouldBeNil)
		listOptions := &metav1.ListOptions{}
		_, err = helper.List("default", listOptions)
		So(err, ShouldBeError)
	})
}

func Test_dynamicCodec(t *testing.T) {
	Convey("test dynamicCodec", t, func() {
		c := dynamicCodec{}
		_, _, err := c.Decode([]byte("abc"), &schema.GroupVersionKind{}, nil)
		So(err, ShouldBeError)
	})
}

func Test_ParseUserIdentity(t *testing.T) {
	Convey("test ParseUserIdentity", t, func() {
		id := "123"
		idencode := base64.StdEncoding.EncodeToString([]byte(id))
		iddecode := ParseUserIdentity(idencode)
		So(id, ShouldEqual, iddecode)
	})
}

func Test_ParseUserGroup(t *testing.T) {
	Convey("test ParseUserGroup", t, func() {
		group := "icp:abc,system:efg"
		groupencode := base64.StdEncoding.EncodeToString([]byte(group))
		groups := ParseUserGroup(groupencode)
		So(len(groups), ShouldEqual, 1)
	})
}
