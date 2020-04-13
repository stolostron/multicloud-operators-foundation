package weave

import "testing"

var testDataJSON = DataJSON{
	Epoch: 1,
	Topologies: map[string]TopologyJSON{
		"pods": {
			Nodes: map[string]NodeJSON{
				"node1;<pod>": {
					Adjacency: []string{""},
					ConnectionLists: []ConnectionListJSON{
						{
							Connections: []ConnectionJSON{
								{NodeID: "node1;<pod>"},
							},
							ID: OutgoingConnectionsID,
						},
						{
							Connections: []ConnectionJSON{
								{NodeID: "node1;<pod>"},
							},
							ID: IncomingConnectionsID,
						},
					},
					ID:       "",
					Label:    "",
					Metadata: []MetadataJSON{},
					Tables: []TableJSON{
						{
							ID: KubLabelsTableID,
							Rows: []TableRowJSON{
								{
									Entries: map[string]string{
										"label": "release",
										"value": "abc",
									},
									ID: ReleaseLabelID,
								},
							},
						},
					},
					Rank: "",
				},
			},
		},
	},
}

func Test_DataJSON(t *testing.T) {
	testDataJSON.AppRelTopology()
}
