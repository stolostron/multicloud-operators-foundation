// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

// Types with "Json" in the name just exist for Unmarshaling the weavescope json. Turns out Unmarshaling json is a little wacky in go, so I made a ton of types.

// Go package management is definitive proof that god does not exist. :)

package weave

import (
	"strings"
)

// String message constants
const (
	AlreadyStartedError        = "WeaveScope data collector already started"
	BodyReadError              = "Error reading weavescope request body"
	CompressError              = "Error compressing weavescope data"
	UnmarshalError             = "Error unmarshaling weave json"
	WeaveRequestError          = "Error sending request to weavescope"
	WeaveCollectorStartMessage = "Weave data collection started"
)

// Constants related to weavescope conventions
const (
	IncomingConnectionsID  = "incoming-connections"
	KubLabelsTableID       = "kubernetes_labels_"
	OutgoingConnectionsID  = "outgoing-connections"
	PodSuffix              = ";<pod>"
	ReleaseLabelID         = "label_release"
	WeaveScopeTopologyPath = "/api/topology"
	KubNamespaceMetadataID = "kubernetes_namespace"
)

// In weavescope every pod has a "tables" section that contains a group of tables.
// One of these tables is the kubernetes_labels_ table, which is the one that is relevant to us.
type TableRowJSON struct {
	Entries map[string]string `json:"entries"`
	ID      string            `json:"id"`
}

type TableJSON struct {
	ID   string         `json:"id"`
	Rows []TableRowJSON `json:"rows"`
}

type ConnectionJSON struct {
	NodeID string `json:"nodeId"`
}

type ConnectionListJSON struct {
	Connections []ConnectionJSON `json:"connections"`
	ID          string           `json:"id"`
}

// There's also an optional "dateType" field, which we ignore because we don't actually care - they're all strings and we only want the namespace one anyway
type MetadataJSON struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Priority int    `json:"priority"`
	Value    string `json:"value"`
}

// A single node in the weavescope pod topology
// Adjacency and ConnectionLists are similar information. Adjacency is a simpler version which is used to format data for mongodb.
type NodeJSON struct {
	Adjacency       []string             `json:"adjacency"` // Ids of adjacent nodes.
	ConnectionLists []ConnectionListJSON `json:"connections"`
	ID              string               `json:"id"`
	Label           string               `json:"label"`
	Metadata        []MetadataJSON       `json:"metadata"`
	Tables          []TableJSON          `json:"tables"`
	Rank            string               `json:"rank"`
}

type TopologyJSON struct {
	Nodes map[string]NodeJSON `json:"nodes"`
}

// The whole json response from weavescope sent to the apiserver.
type DataJSON struct {
	Epoch      int64                   `json:"epoch"`
	Topologies map[string]TopologyJSON `json:"topologies"`
}

func (wdj DataJSON) AppRelTopology() (*Topology, error) {
	hcmtopo := make(Topology)

	// Build the HCM topology out of the weavescope pod topology
	// yikes
	for key, node := range wdj.Topologies["pods"].Nodes { // Find nodes which are pods and have adjacencies
		currentRl := node.ReleaseLabel()
		if strings.HasSuffix(key, PodSuffix) && currentRl != "" { // We only want pods which have at least one adjacency and a release label.
			for _, connList := range node.ConnectionLists {
				addNewEdge(connList, currentRl, hcmtopo)
			}
		}
	}
	return &hcmtopo, nil
}

func addNewEdge(connList ConnectionListJSON, currentRl string, hcmtopo Topology) {
	var weavetopo TopologyJSON
	if connList.ID == OutgoingConnectionsID { // Occasionally things are adjacent to "the internet" or something like that so have to check if it's a pod
		for _, conn := range connList.Connections {
			if destinationNode, exists := weavetopo.Nodes[conn.NodeID]; exists { // If the pod id we just found exists on the other side, we need its release label
				if otherRl := destinationNode.ReleaseLabel(); otherRl != "" && otherRl != currentRl {
					newRel := TopologyEdge{currentRl, otherRl}
					hcmtopo[newRel] = hcmtopo.Has(newRel) // If we already have it we create it with active=true, else active=false
				}
			}
		}
	} else if connList.ID == IncomingConnectionsID {
		for _, conn := range connList.Connections {
			if destinationNode, exists := weavetopo.Nodes[conn.NodeID]; exists { // If the pod id we just found exists on the other side, we need its release label
				if otherRl := destinationNode.ReleaseLabel(); otherRl != "" && otherRl != currentRl {
					// We officially have two connected pods which both have release labels, so add the rel to the hcm topology. But only if it's not already there.
					// Notice here that I put the dest and src in backwards from before. this is for incoming connections, so the "source" is actually the other node.
					newRel := TopologyEdge{otherRl, currentRl}
					hcmtopo[newRel] = hcmtopo.Has(newRel) // If we already have it we create it with active=true, else active=false
				}
			}
		}
	}
}

// Crawls through the newly unmarshalled json object to find the release label, or returns empty string if there is none.
func (nj NodeJSON) ReleaseLabel() string {
	for _, table := range nj.Tables { // Each pod has a list of tables, find the kubernetes labels table.
		if table.ID == KubLabelsTableID {
			for _, row := range table.Rows { // Each table has a list of rows, find the row for the release label.
				if row.ID == ReleaseLabelID { // TODO Not sure syntactically how to make this and the next two if statements into one line
					if label, exists := row.Entries["label"]; exists {
						if label == "release" {
							if value, exists := row.Entries["value"]; exists {
								return value
							}
						}
					}
				}
			}
		}
	}
	// If we got through all the tables and didn't find it, it isn't there.
	return ""
}
