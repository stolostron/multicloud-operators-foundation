package klusterlet

import (
	"context"
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	. "github.com/smartystreets/goconvey/convey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_klusterlet_client(t *testing.T) {
	Convey("test klusterlet_client", t, func() {
		Convey("test klusterlet_client NewClusterConnectionInfoGetter", func() {
			o := NewClientOptions().Config()
			clusterGetter := ClusterGetterFunc(
				func(ctx context.Context, name string, options metav1.GetOptions) (*mcm.ClusterStatus, error) {
					return &mcm.ClusterStatus{}, nil
				})
			_, err := NewClusterConnectionInfoGetter(clusterGetter, o)
			So(err, ShouldBeNil)
		})
		Convey("test klusterlet_client GetConnectionInfo", func() {
			o := NewClientOptions().Config()
			clusterGetter := ClusterGetterFunc(
				func(ctx context.Context, name string, options metav1.GetOptions) (*mcm.ClusterStatus, error) {
					return &mcm.ClusterStatus{}, nil
				})
			getter, _ := NewClusterConnectionInfoGetter(clusterGetter, o)
			_, err := getter.GetConnectionInfo(context.TODO(), "")
			So(err, ShouldBeNil)
		})
	})
}
