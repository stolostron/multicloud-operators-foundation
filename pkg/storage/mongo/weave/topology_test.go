package weave

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTopology(t *testing.T) {
	edge1 := TopologyEdge{Source: "abc", Destination: "efg"}
	edge2 := TopologyEdge{Source: "abc", Destination: "hig"}
	edge3 := TopologyEdge{Source: "efg", Destination: "abc"}

	topology := Topology{
		edge1: true,
		edge2: true,
		edge3: true,
	}

	Convey("test Topology", t, func() {
		Convey("test Has", func() {
			b := topology.Has(edge1)
			So(b, ShouldBeTrue)
		})
		Convey("test HasReverse", func() {
			b := topology.HasReverse(edge3)
			So(b, ShouldBeTrue)
		})
	})
}
