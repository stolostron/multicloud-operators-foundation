package drivers

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func Test_DriverFactory(t *testing.T) {
	Convey("Test DriverFactory", t, func() {
		Convey("Tes DriverFactory New", func() {
			var objects []runtime.Object
			kubeClient := k8sfake.NewSimpleClientset(objects...)
			df := NewDriverFactory(NewDriverFactoryOptions(), kubeClient)
			So(df.logDriver, ShouldNotBeNil)
		})
	})
}
