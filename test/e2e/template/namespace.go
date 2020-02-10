package template

// NamespaceTemplate is json template for namespace
const NamespaceTemplate = `{
    "apiVersion": "v1",
    "kind": "Namespace",
    "metadata": {
        "labels": {
			"test-automation": "true"
		},
        "generateName": "test-automation-"
    }
}`
