package template

// ActionWorkTemplate is json template for action work
const ActionWorkTemplate = `{
    "apiVersion": "mcm.ibm.com/v1beta1",
    "kind": "Work",
    "metadata": {
        "labels": {
			"test-automation": "true"
		},
        "generateName": "deploy-nginx-work-",
        "namespace": "cluster1"
    },
    "spec": {
        "actionType": "Create",
        "cluster": {
            "name": "cluster1"
        },
        "kube": {
            "namespace": "default",
            "resource": "deployment",
            "template": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "metadata": {
                    "labels": {
                        "test-automation": "true"
                    },
                    "generateName": "nginx-deployment-"
                },
                "spec": {
                    "replicas": 1,
                    "selector": {
                        "matchLabels": {
                            "app": "nginx"
                        }
                    },
                    "template": {
                        "metadata": {
                            "labels": {
                                "app": "nginx"
                                }
                            },
                            "spec": {
                                "containers": [
                                    {
                                        "image": "nginx:1.7.9",
                                        "name": "nginx",
                                        "ports": [
                                            {
                                                "containerPort": 80
                                            }
                                        ]
                                    }
                                ]
                            }
                        }
                    }
                }
            },
            "type": "Action"
        }
}`

// ResourceWorkTemplate is the json template for resource work
const ResourceWorkTemplate = `{
    "apiVersion": "mcm.ibm.com/v1beta1",
    "kind": "Work",
    "metadata": {
        "labels": {
			"test-automation": "true"
		},
        "generateName": "fetch-pods-work-",
        "namespace": "cluster1"
    },
    "spec": {
        "cluster": {
            "name": "remote"
        },
        "scope": {
            "resourceType": "pods"
        },
        "type": "Resource"
    }
}`
