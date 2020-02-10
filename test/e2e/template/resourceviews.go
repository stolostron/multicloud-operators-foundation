package template

// ResourceViewTemplate is json template for resourceview
const ResourceViewTemplate = `{
	"apiVersion": "mcm.ibm.com/v1beta1",
	"kind": "ResourceView",
	"metadata": {
        "labels": {
			"test-automation": "true"
		},
		"generateName": "fetch-pods-rv-",
		"namespace": "default"
	},
	"spec": {
		"scope": {
			"resource": "pods"
		}
	}
}`
