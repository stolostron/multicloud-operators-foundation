# Proxy Server Request Configuration

Proxy Server provides a serverless aggregated API `clusterstatus` to proxy requests from managed clusters to other backend servers. The API `clusterstatus` group is `proxy.open-cluster-management.io` and version is `v1beta1`

## clusterstatus/aggregator

Proxy Server provides sub resource under `clusterstatus/aggregator` for client agents on managed clusters to post data to hub.

For example, an agent named search in managed cluster wants to post data to its backend search service `<service-host>:<port>/search/cluster/<cluster-name>/<sub resource>/xxx`.
It can post data to `apis/proxy.open-cluster-management.io/v1beta1/namespaces/<cluster-namespace>/clusterstatuses/<cluster-name>/aggregator/<sub resource>/xxx`.
Proxy Server will proxy the requests to the backend search service who finally save the data.

### Configuration Example

User can configure the proxy information using a configMap with the label `config: acm-proxyserver`.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: search-proxy
  labels:
    config: acm-proxyserver
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

## clusterstatus/log

Proxy Server provides sub resource under `clusterstatus/log` for clients to get logs of container on managed clusters.

For example, user can GET `apis/proxy.open-cluster-management.io/v1beta1/namespaces/cluster0/clusterstatuses/cluster0/log/multicloud-system/mongo-0/mongo` if we want to get logs of container `mongo` in pod `mongo-0` of namespace `default` in managed cluster `cluster0`.