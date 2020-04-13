// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package weave

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMongo(t *testing.T) {
	Convey("Test Mongo", t, func() {
		node1 := NodeJSON{
			Adjacency:       []string{"abc", "efg"},
			ConnectionLists: []ConnectionListJSON{},
			ID:              "abc",
			Label:           "label1",
			Metadata:        []MetadataJSON{},
			Tables:          []TableJSON{},
			Rank:            "rank1",
		}
		topology := TopologyJSON{Nodes: map[string]NodeJSON{"abc": node1}}

		topology.Mongo()
	})
}

func Test_NodeJSON_UID(t *testing.T) {
	Convey("Test NodeJSON: UID", t, func() {
		testCases := []struct {
			name string
			nj   NodeJSON
			rst  string
		}{
			{
				name: "UDI case1",
				nj:   NodeJSON{ID: "internet123"},
				rst:  "internet123",
			},
			{
				name: "UDI case2",
				nj:   NodeJSON{ID: "123;abc;efg"},
				rst:  "123",
			},
			{
				name: "UDI case3",
				nj:   NodeJSON{ID: "123:abc:efg"},
				rst:  "efg",
			},
		}
		for _, testCase := range testCases {
			So(testCase.nj.UID(), ShouldEqual, testCase.rst)
		}
	})
}

func Test_NodeJSON_Type(t *testing.T) {
	Convey("Test NodeJSON: Type", t, func() {
		testCases := []struct {
			name string
			nj   NodeJSON
			rst  string
		}{
			{
				name: "Type case1",
				nj:   NodeJSON{ID: "internet123"},
				rst:  "internet",
			},
			{
				name: "Type case2",
				nj:   NodeJSON{ID: "abc<12345>efg"},
				rst:  "12345",
			},
			{
				name: "Type case3",
				nj:   NodeJSON{ID: "123:abc:efg"},
				rst:  "unmanaged",
			},
		}
		for _, testCase := range testCases {
			So(testCase.nj.Type(), ShouldEqual, testCase.rst)
		}
	})
}

func Test_NodeJSON_Namespace(t *testing.T) {
	Convey("Test NodeJSON: Namespace", t, func() {
		testCases := []struct {
			name string
			nj   NodeJSON
			rst  string
		}{
			{
				name: "Namespace case1",
				nj:   NodeJSON{Rank: "rank"},
				rst:  "rank",
			},
			{
				name: "Namespace case2",
				nj:   NodeJSON{Rank: "rank/abc"},
				rst:  "rank",
			},
		}
		for _, testCase := range testCases {
			So(testCase.nj.Namespace(), ShouldEqual, testCase.rst)
		}
	})
}

func Test_NodeJSON_MongoLabels(t *testing.T) {
	Convey("Test NodeJSON: MongoLabels", t, func() {
		testCases := []struct {
			name string
			nj   NodeJSON
			rst  []MongoLabel
		}{
			{
				name: "MongoLabels case1",
				nj: NodeJSON{
					Tables: []TableJSON{
						{
							ID: "kubernetes_labels_",
							Rows: []TableRowJSON{
								{
									Entries: map[string]string{
										"label": "abc",
										"value": "123",
									},
								},
							},
						},
						{
							ID:   "kubernetes_labels_",
							Rows: []TableRowJSON{},
						},
					},
				},
				rst: []MongoLabel{
					{
						Name:  "abc",
						Value: "123",
					},
				},
			},
		}
		for _, testCase := range testCases {
			So(testCase.nj.MongoLabels()[0].Name, ShouldEqual, testCase.rst[0].Name)
			So(testCase.nj.MongoLabels()[0].Value, ShouldEqual, testCase.rst[0].Value)
		}
	})
}

func Test_DataJSON_Mongo(t *testing.T) {
	Convey("Test DataJSON: Mongo", t, func() {
		wdj := DataJSON{
			Epoch: 1,
			Topologies: map[string]TopologyJSON{
				"topoType": {
					Nodes: map[string]NodeJSON{
						"node1": {
							ID:   "123;123",
							Rank: "abc",
							Tables: []TableJSON{
								{
									ID: "kubernetes_labels_",
									Rows: []TableRowJSON{
										{
											Entries: map[string]string{
												"label": "abc",
												"value": "123",
											},
										},
									},
								},
							},
							Adjacency: []string{"node1"},
						},
					},
				},
			},
		}
		wdj.Mongo()
	})
}

func Test_MarshalCompressEncodeArray(t *testing.T) {
	Convey("Test Test_MarshalCompressEncodeArray", t, func() {
		mongoResourceData := []MongoResource{
			{
				Cluster:   "cluster1",
				Epoch:     1,
				Name:      "cluster1",
				Namespace: "cluster1",
				UID:       "abc",
			},
			{
				Cluster:   "cluster2",
				Epoch:     1,
				Name:      "cluster2",
				Namespace: "cluster2",
				UID:       "123",
			},
		}

		MarshalCompressEncodeArray(mongoResourceData)
	})
}
