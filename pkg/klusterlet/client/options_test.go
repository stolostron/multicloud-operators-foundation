package klusterlet

import (
	"errors"
	"testing"

	"bou.ke/monkey"
	. "github.com/smartystreets/goconvey/convey"
	certutil "k8s.io/client-go/util/cert"
)

func Test_Options(t *testing.T) {
	Convey("test client options", t, func() {
		Convey("test clint options MaybeDefaultWithSelfSignedCerts case1", func() {
			options := NewClientOptions()
			options.CAFile = "/tmp"
			err := options.MaybeDefaultWithSelfSignedCerts("1.1.1.1")
			So(err, ShouldBeNil)
		})
		Convey("test clint options MaybeDefaultWithSelfSignedCerts case2", func() {
			options := NewClientOptions()
			err := options.MaybeDefaultWithSelfSignedCerts("1.1.1.1")
			So(err, ShouldBeNil)
		})
		Convey("test clint options MaybeDefaultWithSelfSignedCerts case3", func() {
			defer monkey.UnpatchAll()
			options := NewClientOptions()
			monkey.Patch(certutil.CanReadCertAndKey, func(certPath, keyPath string) (bool, error) {
				return false, errors.New("fail to CanReadCertAndKey")
			})
			err := options.MaybeDefaultWithSelfSignedCerts("1.1.1.1")
			So(err, ShouldBeError)
		})
		Convey("test clint options MaybeDefaultWithSelfSignedCerts case4", func() {
			defer monkey.UnpatchAll()
			options := NewClientOptions()
			monkey.Patch(certutil.CanReadCertAndKey, func(certPath, keyPath string) (bool, error) {
				return false, nil
			})
			err := options.MaybeDefaultWithSelfSignedCerts("1.1.1.1")
			So(err, ShouldBeNil)
		})
		Convey("test clint options Config case1", func() {
			options := NewClientOptions()
			c := options.Config()
			So(c.CertFile, ShouldEqual, options.CertFile)
		})
	})
}
