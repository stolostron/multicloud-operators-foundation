package template

// ClusterJoinRequestTemplate is json template for namespace
const ClusterJoinRequestTemplate = `{
	"apiVersion": "mcm.ibm.com/v1beta1",
	"kind": "ClusterJoinRequest",
	"metadata": {
		"labels": {
			"test-automation": "true"
		},
		"generateName": "clusterjoinrequest-"
	},
	"spec": {
		"clusterName": "cluster1",
		"clusterNameSpace": "cluster1",
		"csr": {
			"request": "",
			"usages": [
				"digital signature",
				"key encipherment",
				"client auth"
			]
		}
	}
}`
