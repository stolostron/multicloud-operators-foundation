package errors

import (
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_NoClusterSpecifiedError(t *testing.T) {
	Convey("test NoClusterSpecifiedError", t, func() {
		Convey("test NoClusterSpecifiedError case1", func() {
			err := NewNoClusterError()
			msg := err.Error()
			So(msg, ShouldEqual, "No cluster specified")
		})
		Convey("test NoClusterSpecifiedError case2", func() {
			err := errors.New("no cluster specified")
			rst := IsNoClusterError(err)
			So(rst, ShouldBeFalse)
		})
	})
}

func Test_AssetSecretNotFoundError(t *testing.T) {
	Convey("test AssetSecretNotFoundError", t, func() {
		Convey("test AssetSecretNotFoundError case1", func() {
			err := NewAssetSecretNotFoundError("foo", "default")
			msg := err.Error()
			So(msg, ShouldEqual, "Secret foo not found in namespace default")
		})
		Convey("test AssetSecretNotFoundError case2", func() {
			err := errors.New("no cluster specified")
			rst := IsAssetSecretNotFoundError(err)
			So(rst, ShouldBeFalse)
		})
	})
}
