package template

// ClusterTemplate is json template for namespace
const ClusterTemplate = `{
	"apiVersion": "clusterregistry.k8s.io/v1alpha1",
	"kind": "Cluster",
	"metadata": {
		"labels": {
			"test-automation": "true"
		},
		"name": "cluster1",
		"namespace": "cluster1"
	},
	"spec": {
	}
}`
