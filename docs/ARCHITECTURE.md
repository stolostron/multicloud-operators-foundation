# Architecture

## Purpose

`multicloud-operators-foundation` supplies ACM foundation services on the Hub and
managed clusters. The repository builds two long-running Go processes: a Hub-side
foundation controller and a managed-cluster work-manager agent.

## Hub-side controller

`cmd/controller` creates a controller-runtime manager against the Hub cluster and
registers the reconcilers under `pkg/controllers`. The controllers watch ACM,
Kubernetes, Hive, OpenShift, and managed-service-account resources and use typed
clients, informers, and shared caches to reconcile:

- Managed cluster information, cluster sets, cluster claims, and deployments.
- Cluster-set mappings and synchronized role bindings for RBAC propagation.
- Image registry configuration, add-on deployment, managed service accounts,
  garbage collection, and cluster role/CA resources.

The controller uses leader election when enabled. Health and readiness probes are
served by controller-runtime on port `8000`. The add-on manager optionally installs
the work-manager agent through the add-on framework and Helm chart in `pkg/addon`.

## Managed-cluster agent

`cmd/agent` connects to the Hub, management cluster, and managed cluster using
separate Kubernetes client configurations where required. It runs a controller-
runtime manager for Hub-scoped resources and shared client-go informers for managed
cluster resources. The agent registers reconcilers under `pkg/klusterlet` for:

- `ManagedClusterAction`, applying requested operations to the managed cluster.
- `ManagedClusterView`, reading arbitrary managed-cluster resources for the Hub.
- `ManagedClusterInfo` and cluster claims, publishing cluster state and metadata.
- Node resource collection and managed-service-account related behavior.

The agent also maintains leases and serves health/config probes on port `8000`.
Its clients use a reloadable REST mapper because managed-cluster API discovery can
change over time.

## Data flow and boundaries

1. Hub controllers observe ACM and Kubernetes resources on the Hub.
2. Hub reconcilers create or update add-on and control-plane resources, including
   the work-manager deployment configuration.
3. The agent observes Hub requests and managed-cluster state through its respective
   clients and informers.
4. Agent reconcilers execute managed-cluster actions or views and publish results,
   claims, node data, and health state back through ACM APIs.
5. Generated APIs and CRDs are maintained by `hack/` scripts and validated by
   `make verify`.

The Hub controller and agent are intentionally separate processes: Hub-side
reconciliation must not depend on direct managed-cluster access, while agent-side
operations need credentials and API discovery scoped to the managed cluster.

## Testing and deployment

Unit tests live alongside packages. Envtest integration tests are under
`test/integration`; end-to-end tests under `test/e2e` require deployed Hub and
managed-cluster components. Deployment and generated resources are under `deploy`.
The Makefile delegates common build and image behavior to
`openshift/build-machinery-go`.
