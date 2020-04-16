// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

// Set of types and functions for use in unmarshaling/restructuring weavescope data which needs to be put into mongodb.
// A little bit weird because the functions are methods on types that are defined in weavescoperestmonitor.go
// But, they are all super related so I thought it was cleaner to have them all be in a file together.

package weave

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"strings"
)

const (
	GzipCompressionLevel = gzip.BestCompression
)

type MongoLabel struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MongoRelationship is just a MongoResource without a relationships field. These go in the relationships list.
// They don't have a relationships field because the data isn't cyclic
type MongoRelationship struct {
	Cluster   string       `json:"cluster"` // TODO This is some sort of unique mongo ID thing. May not actually be a string.
	Labels    []MongoLabel `json:"labels"`
	Name      string       `json:"name"`
	Namespace string       `json:"namespace"`
	Topology  string       `json:"topology"` // A supported topology type from weavescope
	Type      string       `json:"type"`
	UID       string       `json:"uid"`
}

// MongoResource is what you marshal into.
type MongoResource struct {
	Cluster       string              `json:"cluster"` // TODO This is some sort of unique mongo ID thing. May not actually be a string.
	Epoch         int64               `json:"epoch"`
	Labels        []MongoLabel        `json:"labels"`
	Name          string              `json:"name"`
	Namespace     string              `json:"namespace"`
	Relationships []MongoRelationship `json:"relationships"` // Destinations for edges originating at this resource
	Topology      string              `json:"topology"`      // A supported topology type from weavescope
	Type          string              `json:"type"`
	UID           string              `json:"uid"`
}

// Gets UID of a given NodeJSON splitting from its .ID
func (nj NodeJSON) UID() string {
	if strings.Contains(nj.ID, "internet") {
		return nj.ID
	}
	if strings.Contains(nj.ID, ";") {
		return nj.ID[0:strings.Index(nj.ID, ";")]
	}
	if strings.Contains(nj.ID, ":") {
		return strings.Split(nj.ID, ":")[2]
	}
	return nj.ID
}

// Gets Type of a given NodeJSON splitting from its .ID
func (nj NodeJSON) Type() string {
	if strings.Contains(nj.ID, "internet") {
		return "internet"
	}
	if strings.Contains(nj.ID, "<") && strings.Contains(nj.ID, ">") {
		return nj.ID[strings.Index(nj.ID, "<")+1 : strings.Index(nj.ID, ">")]
	}
	return "unmanaged"
}

// Get Namespace of a given NodeJSON splitting from its .Rank
func (nj NodeJSON) Namespace() string {
	if !strings.Contains(nj.Rank, "/") {
		return nj.Rank
	}
	return nj.Rank[0:strings.Index(nj.Rank, "/")]
}

// Crawls through the newly unmarshalled json object to find the release label, or returns empty string if there is none.
func (nj NodeJSON) MongoLabels() []MongoLabel {
	var ret []MongoLabel
	for _, table := range nj.Tables { // Each pod has a list of tables, find the kubernetes labels table.
		if table.ID == KubLabelsTableID {
			for _, row := range table.Rows { // Each table has a list of rows.
				if label, exists := row.Entries["label"]; exists {
					if value, exists := row.Entries["value"]; exists {
						ret = append(ret, MongoLabel{label, value})
					}
				}
			}
		}
	}
	return ret
}

// Returns the complete list of resources present in a set of Weave data in the format that Mongo wants them.
// Leaves out cluster field because it's unknown to the DataJSON object itself
func (wdj DataJSON) Mongo() []MongoResource {
	var ret []MongoResource
	for topoType, tj := range wdj.Topologies { // they're keyed by the topology type e.g. pods, containers, hosts
		add := tj.Mongo()
		for _, mr := range add { // add in all the Topology fields, which were unknown to the functions lower than this.
			mr.Topology = topoType
			mr.Epoch = wdj.Epoch // Fill in from Epoch on the whole response so that all the resources match
		}
		ret = append(ret, add...)
	}
	return ret
}

// Returns MongoResource values for all the nodes in a weavescope topology.
// Leaves a couple fields blank, because they're unknown to the TopologyJSON object itself. Will fill them out in the caller to this.
func (tj TopologyJSON) Mongo() []MongoResource {
	var ret []MongoResource
	for _, node := range tj.Nodes { // Find nodes which are pods and have adjacencies
		add := MongoResource{"", 0, node.MongoLabels(), node.Label, node.Namespace(), []MongoRelationship{}, "", node.Type(), node.UID()}
		for _, adjacent := range node.Adjacency {
			if adjNode, ok := tj.Nodes[adjacent]; ok {
				add.Relationships = append(add.Relationships, adjNode.MongoRelationship())
			}
		}
		ret = append(ret, add)
	}
	return ret
}

// Leaves a couple fields blank, because they're unknown to the NodeJSON object itself. Will fill them out in the caller to this.
func (nj NodeJSON) MongoRelationship() MongoRelationship {
	return MongoRelationship{"", nj.MongoLabels(), nj.Label, nj.Namespace(), "", nj.Type(), nj.UID()}
}

// Marhshals/compress/encodes an entire array of mongoresources.
func MarshalCompressEncodeArray(mongoResourceData []MongoResource) (string, error) {
	marshalledJSON, err := json.Marshal(mongoResourceData)
	if err != nil {
		return "", err
	}

	var compressed bytes.Buffer
	w, err := gzip.NewWriterLevel(&compressed, GzipCompressionLevel)
	if err != nil {
		return "", err
	}

	_, err = w.Write(marshalledJSON)
	if err != nil {
		return "", err
	}
	err = w.Close()
	if err != nil {
		return "", err
	}

	encodedData := base64.StdEncoding.EncodeToString(compressed.Bytes())
	return encodedData, nil
}
