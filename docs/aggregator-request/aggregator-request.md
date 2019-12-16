# Aggregator Request Configuration

Multicloud manager provides subresource under `clusterstatus/aggregator` for client agents to post data to hub.

For example, An agent named search in managed cluster wants to post data to its backend search service `<service-host>:<port>/search/cluster/<cluster-name>/<sub resource>/xxx`.
It can post data to `apis/mcm.ibm.com/v1alpha1/namespaces/<cluster-namespace>/clusterstatuses/<cluster-name>/aggregator/<sub resource>/xxx`.
The aggregated API server will proxy the requests to the backend search service who finally save the data.

The value of using aggregated api as a proxy is:
1. The kube apiserver can do the authentication for all requests from agents. The agents do not need additional credential to connect to hub server.
They can share the same `kubeconfig` with other agents in klusterlet.
2. Aggregated apisever can do the authorization on agent requests from different clusters.

## Configuration Example

User can configure the proxy information using a configMap with the label `config: mcm-aggregator`.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: search-proxy
  labels:
    config: mcm-aggregator
data:
  service: "kube-system/service-name"
  port: "8801"
  path: "/search/cluster/"
  sub-resource: "/searchdata"
  use-id: "true"
  secret: "kube-system/search-secret"
```

**data:**

* `service`: The backend service name, Format is `namespace/<service-name>`.
* `port`: The export port of the backend service.
* `path`: The path for agents to send requests.
* `sub-resource`: The resource of requests from agents.
* `use-id`: If true, the path of requests includes cluster name as ID. If false, there is no ID in the path.
* `secret`: The secret with CA information to access the backend service. Format is `namespace/<secret-name>`.
