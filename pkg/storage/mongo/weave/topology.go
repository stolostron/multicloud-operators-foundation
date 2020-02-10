// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

// This package is used for keeping live topology data, which is sent back to the hcm-application component to be stored in mongodb.

package weave

const (
	DefaultWeaveScopeParamsString = "pseudo=hide&namespace=&t=5s" // params for weaveScope params
)

// TODO
// A type representing the entire set of topology data.
// Currently, this is only ever used to determine whether 2 helm releases are connected, we never have to ask whether a helm release exists, etc.
// This means that even though it's a graph, we don't have to store anything about the nodes, only edges.

// A map of edge pairs. The value here is true/false, false is inactive and true is active.
// Really there are 3 states here - active, inactive, and nonexistent.
type Topology map[TopologyEdge]bool

// A pair of strings representing the source/dest of the relationship.
// The status field used to be kept in here but is now the value part of a map in the Topology type.
type TopologyEdge struct {
	Source      string // Helm release name of source node
	Destination string // Helm release name of destination node
}

// Topology monitors need to be able to return the current topology, and that's all they have to do.
type TopologyMonitor interface {
	CurrentTopology() (*Topology, error)
}

// Checks whether one TopologyRelationship and another have same source and destination (does not check the status)
func (one TopologyEdge) SamePair(two TopologyEdge) bool {
	return (one.Source == two.Source && one.Destination == two.Destination)
}

// Checks whether one TopologyRelationship and another are the same but swapped src/dest
func (one TopologyEdge) ReversePair(two TopologyEdge) bool {
	return (one.Source == two.Destination && one.Destination == two.Source)
}

// Whether or not the Topology has the given TopolgyEdge already.
func (t Topology) Has(tr TopologyEdge) bool {
	for rel := range t {
		if rel.SamePair(tr) {
			return true
		}
	}

	return false
}

// Whether or not the Topology has the reverse of the given TopologyRelationship already.
func (t Topology) HasReverse(tr TopologyEdge) bool {
	for rel := range t {
		if rel.ReversePair(tr) {
			return true
		}
	}
	return false
}
