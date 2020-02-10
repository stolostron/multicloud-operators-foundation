package template

// WorksetsTemplate is json template for namespace
const WorksetsTemplate = `{
	"apiVersion": "mcm.ibm.com/v1beta1",
	"kind": "WorkSet",
	"metadata": {
        "labels": {
			"test-automation": "true"
		},
        "generateName": "create-configmap-workset-",
        "namespace": "default"
	},
	"spec": {
		"clusterSelector": {},
		"template": {
			"spec": {
				"actionType": "Create",
				"kube": {
                    "namespace": "default",
					"resource": "configmap",
					"template": {
						"apiVersion": "v1",
						"data": {
							"redis-config": "maxmemory 2mb\nmaxmemory-policy allkeys-lru\n"
						},
						"kind": "ConfigMap",
						"metadata": {
                            "labels": {
                                "test-automation": "true"
                            },
                            "generateName": "redis-config-",
                            "namespace": "default"
						}
					}
				},
				"type": "Action"
			}
		}
	}
}`
