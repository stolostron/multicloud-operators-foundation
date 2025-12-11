# ClusterView API Performance Testing

This directory contains performance testing tools for ClusterView aggregated APIs.

## Available Performance Tests

This directory provides performance testing for two ClusterView aggregated APIs:

- **KubeVirt Projects API** (`kubevirtprojects.clusterview.open-cluster-management.io`) - Aggregates project access information across managed clusters
- **UserPermission API** (`userpermissions.clusterview.open-cluster-management.io`) - Discovers user permissions across the fleet of managed clusters

Each API has its own dedicated performance test script and documentation below.

## Prerequisites

### Mock ManagedCluster Setup

Both performance test scripts require existing managed cluster namespaces. You have two options:

#### Option 1: Use Existing Managed Clusters (Recommended for Production Environments)

If you already have managed clusters in your environment, the test scripts will automatically discover and use them.

#### Option 2: Create Mock Clusters for Testing (Recommended for Development)

Use the `setup-mock-clusters.sh` script to create mock ManagedCluster resources without actual Kubernetes clusters:

```bash
# Create 10 mock clusters (default)
./setup-mock-clusters.sh setup

# Create custom number of clusters
NUM_CLUSTERS=20 ./setup-mock-clusters.sh setup

# List mock clusters
./setup-mock-clusters.sh list

# Cleanup after testing
./setup-mock-clusters.sh cleanup
```

**Benefits of mock clusters:**
- Fast setup/teardown (seconds vs minutes)
- No Docker or additional resources needed
- Perfect for CI/CD and local development
- Isolated from production clusters

See [Mock Cluster Setup](#mock-cluster-setup) section below for detailed documentation.

---

## KubeVirt Projects API Performance Testing

Script: `test-kubevirtprojects-performance.sh`

### Overview

The `kubevirtprojects` API aggregates project access information across multiple managed clusters based on ClusterPermission objects. This test script helps measure performance under various conditions.

**Key Feature:** Creates **multiple ClusterPermission objects** per cluster namespace to better simulate real-world scenarios and test indexer performance.

### Prerequisites

1. **kubectl** - Configured and connected to your test cluster
2. **jq** - JSON processor for parsing API responses
3. **bc** - Calculator for performance metrics
4. **ClusterPermission CRD** - Installed in the cluster
5. **kubevirtprojects API** - Available and accessible

#### Installing Prerequisites

```bash
# macOS
brew install jq bc

# Linux (Ubuntu/Debian)
sudo apt-get install jq bc

# Linux (RHEL/CentOS)
sudo yum install jq bc
```

### Quick Start

```bash
# Run full test suite with defaults
KUBECONFIG=/path/to/kubeconfig bash test-kubevirtprojects-performance.sh full

# Custom scale test
KUBECONFIG=/path/to/kubeconfig \
NUM_CLUSTERS=5 \
PROJECTS_PER_CLUSTER=50 \
NUM_CLUSTERPERMISSIONS=10 \
bash test-kubevirtprojects-performance.sh full
```

### Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NUM_CLUSTERS` | Number of cluster namespaces to use | 10 |
| `PROJECTS_PER_CLUSTER` | Total projects per cluster | 10 |
| `NUM_CLUSTERPERMISSIONS` | ClusterPermission objects per cluster | 5 |
| `NUM_ITERATIONS` | Performance test iterations | 10 |
| `KUBECONFIG` | Path to kubeconfig file | ~/.kube/config |

### How It Works

#### Multiple ClusterPermissions Distribution

The script distributes projects across multiple ClusterPermission objects to test indexer performance.

**Example: `NUM_CLUSTERPERMISSIONS=5`, `PROJECTS_PER_CLUSTER=10`**

In cluster namespace `cluster-1`, creates 5 ClusterPermissions:

```
ClusterPermission: perf-test-cp-1
  - project-1, project-2

ClusterPermission: perf-test-cp-2
  - project-3, project-4

ClusterPermission: perf-test-cp-3
  - project-5, project-6

ClusterPermission: perf-test-cp-4
  - project-7, project-8

ClusterPermission: perf-test-cp-5
  - project-9, project-10
```

**Total:** 5 ClusterPermissions per cluster × 10 clusters = **50 ClusterPermission objects**

#### What Gets Tested

The API performance for:

1. **Index lookup**: Finding all ClusterPermissions for the user/group
2. **Iteration**: Processing each ClusterPermission object
3. **Extraction**: Getting roleBindings from each ClusterPermission
4. **Aggregation**: Combining projects from multiple ClusterPermissions
5. **Deduplication**: Removing duplicate projects (using sets)
6. **Sorting**: Sorting results by cluster name
7. **Generation**: Creating PartialObjectMetadata for each project

### Commands

#### `setup` - Create Test Data

Creates ClusterPermissions in existing cluster namespaces:

```bash
KUBECONFIG=/path/to/config bash test-kubevirtprojects-performance.sh setup
```

This creates:

- Multiple ClusterPermission objects per cluster
- RBAC for test user/group
- Test projects across all clusters

#### `test` - Run Performance Test

Measures API response times:

```bash
KUBECONFIG=/path/to/config NUM_ITERATIONS=100 bash test-kubevirtprojects-performance.sh test
```

Output includes:

- Response time per iteration
- Min/max/average response times
- Number of projects returned
- Throughput (requests/second)

#### `full` - Complete Test Suite

Runs setup → test → cleanup:

```bash
KUBECONFIG=/path/to/config bash test-kubevirtprojects-performance.sh full
```

#### `cleanup` - Remove Test Data

Removes all test ClusterPermissions:

```bash
KUBECONFIG=/path/to/config bash test-kubevirtprojects-performance.sh cleanup
```

### Test Scenarios

#### Scenario 1: Indexer Stress Test

Test with many ClusterPermission objects:

```bash
NUM_CLUSTERS=5 \
PROJECTS_PER_CLUSTER=100 \
NUM_CLUSTERPERMISSIONS=20 \
NUM_ITERATIONS=50 \
bash test-kubevirtprojects-performance.sh full
```

**Result:** 100 ClusterPermissions (5 clusters × 20 CPs each), 500 total projects

#### Scenario 2: Large ClusterPermissions

Test with fewer but larger ClusterPermission objects:

```bash
NUM_CLUSTERS=10 \
PROJECTS_PER_CLUSTER=100 \
NUM_CLUSTERPERMISSIONS=2 \
NUM_ITERATIONS=50 \
bash test-kubevirtprojects-performance.sh full
```

**Result:** 20 ClusterPermissions (10 × 2), each with 50 projects

#### Scenario 3: Balanced Distribution

Moderate and balanced test:

```bash
NUM_CLUSTERS=10 \
PROJECTS_PER_CLUSTER=50 \
NUM_CLUSTERPERMISSIONS=5 \
NUM_ITERATIONS=100 \
bash test-kubevirtprojects-performance.sh full
```

**Result:** 50 ClusterPermissions, 500 total projects

#### Scenario 4: High ClusterPermission Count

Maximum ClusterPermission objects:

```bash
NUM_CLUSTERS=3 \
PROJECTS_PER_CLUSTER=60 \
NUM_CLUSTERPERMISSIONS=30 \
NUM_ITERATIONS=50 \
bash test-kubevirtprojects-performance.sh full
```

**Result:** 90 ClusterPermissions (3 × 30), 2 projects per CP

### Sample Output

```
[INFO] Creating 5 ClusterPermissions for cluster-1 (50 projects total)...
[INFO]   Created 5 ClusterPermissions for cluster-1
[INFO] Setup complete!
[INFO] Total ClusterPermissions created: 50
[INFO] Total projects: 500

[INFO] Measuring API performance with 50 iterations...

Iteration | Response Time (ms) | Projects Returned
----------|-------------------|------------------
        1 |               245 |               500
        2 |               198 |               500
        3 |               203 |               500
...

[INFO] Performance Summary:
  Average response time: 215 ms
  Min response time: 198 ms
  Max response time: 289 ms
  Projects returned: 500
  Throughput: 4.65 requests/second
```

### Performance Recommendations

#### Development Testing

```bash
NUM_CLUSTERS=2 NUM_CLUSTERPERMISSIONS=3 PROJECTS_PER_CLUSTER=10
```

**6 ClusterPermissions, 20 projects** - Quick feedback

#### Realistic Testing

```bash
NUM_CLUSTERS=10 NUM_CLUSTERPERMISSIONS=5 PROJECTS_PER_CLUSTER=50
```

**50 ClusterPermissions, 500 projects** - Realistic scale

#### Performance Benchmarking

```bash
NUM_CLUSTERS=20 NUM_CLUSTERPERMISSIONS=10 PROJECTS_PER_CLUSTER=100
```

**200 ClusterPermissions, 2000 projects** - Stress test

#### Indexer Stress Test

```bash
NUM_CLUSTERS=10 NUM_CLUSTERPERMISSIONS=50 PROJECTS_PER_CLUSTER=100
```

**500 ClusterPermissions** - Tests indexer specifically

### Expected Performance

Baseline expectations (depends on cluster resources):

| Scale | Clusters | Projects | ClusterPermissions | Expected Response |
|-------|----------|----------|-------------------|------------------|
| Small | 5 | 50 | 25 | < 100ms |
| Medium | 10 | 500 | 50 | < 500ms |
| Large | 20 | 2000 | 200 | < 1s |
| Very Large | 50 | 5000 | 500 | < 3s |

**Good Performance Indicators:**

- Response time scales linearly with data size
- Min/max times within 2x of average
- No timeout errors
- Consistent throughput across iterations

### Troubleshooting

#### API Not Found

Check if the API server is running and CRD is installed:

```bash
kubectl get crd clusterpermissions.rbac.open-cluster-management.io
```

#### Permission Errors

Verify RBAC is created:

```bash
kubectl get clusterrolebinding test-user-kubevirtprojects
kubectl get clusterrole kubevirtprojects-viewer -o yaml
```

#### Slow Performance

1. Check API server resources:

   ```bash
   kubectl top pods -n kube-system
   ```

2. Review API server logs:

   ```bash
   kubectl logs -n kube-system -l component=kube-apiserver --tail=100
   ```

3. Reduce test scale:

   ```bash
   NUM_CLUSTERS=5 NUM_CLUSTERPERMISSIONS=3 bash test-kubevirtprojects-performance.sh setup
   ```

#### Out of Memory

- Reduce `NUM_CLUSTERS` or `NUM_CLUSTERPERMISSIONS`
- Check for memory leaks in the implementation
- Increase API server memory limits

### Performance Optimization Tips

Based on test results, consider:

1. **Caching**: Implement caching for ClusterPermission lookups
2. **Indexing**: Ensure proper indexing on subject names/groups
3. **Pagination**: Add support for pagination with large result sets
4. **Filtering**: Allow filtering by cluster to reduce result size
5. **Concurrent Processing**: Process multiple clusters concurrently
6. **Watch API**: Use watch API instead of repeated polls

### Next Steps

After gathering performance data:

1. Compare results with different ClusterPermission distributions
2. Identify bottlenecks (indexing vs iteration vs aggregation)
3. Profile the API server if needed
4. Consider implementing optimizations listed above

### References

- API Implementation: [pkg/proxyserver/rest/project/rest.go](../../pkg/proxyserver/rest/project/rest.go)
- Project Listing: [pkg/proxyserver/rest/project/list.go](../../pkg/proxyserver/rest/project/list.go)
- ClusterPermission Indexing: [pkg/proxyserver/rest/project/index.go](../../pkg/proxyserver/rest/project/index.go)

---

## UserPermission API Performance Testing

Script: `test-userpermission-performance.sh`

### Overview

The `userpermissions` API allows users to discover their permissions across the fleet of managed clusters. It returns ClusterRole definitions and their cluster/namespace bindings based on ClusterPermission resources and discoverable ClusterRoles.

**Key Features:**

- Tests both LIST and GET operations
- Creates **multiple discoverable ClusterRoles** with the label `clusterview.open-cluster-management.io/discoverable: "true"`
- Creates **multiple ClusterPermission objects** per cluster namespace referencing the discoverable ClusterRoles
- Tests caching layer performance

### Prerequisites

Same as kubevirtprojects API:

1. **kubectl** - Configured and connected to your test cluster
2. **jq** - JSON processor for parsing API responses
3. **bc** - Calculator for performance metrics
4. **ClusterPermission CRD** - Installed in the cluster
5. **userpermissions API** - Available and accessible

### Quick Start

```bash
# Run full test suite with defaults
KUBECONFIG=/path/to/kubeconfig bash test-userpermission-performance.sh full

# Custom scale test
KUBECONFIG=/path/to/kubeconfig \
NUM_CLUSTERS=5 \
NAMESPACES_PER_CLUSTER=20 \
NUM_CLUSTERPERMISSIONS=10 \
NUM_CLUSTERROLES=10 \
bash test-userpermission-performance.sh full
```

### Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NUM_CLUSTERS` | Number of cluster namespaces to use | 10 |
| `NAMESPACES_PER_CLUSTER` | Total namespace bindings per cluster | 10 |
| `NUM_CLUSTERPERMISSIONS` | ClusterPermission objects per cluster | 5 |
| `NUM_CLUSTERROLES` | Discoverable ClusterRoles to create | 5 |
| `NUM_ITERATIONS` | Performance test iterations | 10 |
| `KUBECONFIG` | Path to kubeconfig file | ~/.kube/config |

### How It Works

#### Discoverable ClusterRoles

The script creates ClusterRoles with the discoverable label:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: perf-test-clusterrole-1
  labels:
    clusterview.open-cluster-management.io/discoverable: "true"
    test: "performance"
rules:
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "create", "update", "delete"]
```

#### ClusterPermissions Distribution

**Example: `NUM_CLUSTERS=2`, `NUM_CLUSTERPERMISSIONS=3`, `NAMESPACES_PER_CLUSTER=6`, `NUM_CLUSTERROLES=2`**

Creates 3 ClusterPermissions per cluster, referencing the 2 discoverable ClusterRoles:

```
cluster-1:
  ClusterPermission: perf-test-cp-1 (refs perf-test-clusterrole-1)
    - namespace-1, namespace-2
  ClusterPermission: perf-test-cp-2 (refs perf-test-clusterrole-2)
    - namespace-3, namespace-4
  ClusterPermission: perf-test-cp-3 (refs perf-test-clusterrole-1)
    - namespace-5, namespace-6

cluster-2:
  ClusterPermission: perf-test-cp-1 (refs perf-test-clusterrole-1)
    - namespace-1, namespace-2
  ClusterPermission: perf-test-cp-2 (refs perf-test-clusterrole-2)
    - namespace-3, namespace-4
  ClusterPermission: perf-test-cp-3 (refs perf-test-clusterrole-1)
    - namespace-5, namespace-6
```

**Total:**

- 2 discoverable ClusterRoles
- 6 ClusterPermissions (2 clusters × 3 CPs each)
- 12 namespace bindings total

#### What Gets Tested

The API performance for:

**LIST Operation:**

1. **ClusterRole Discovery**: Finding all discoverable ClusterRoles
2. **ClusterPermission Lookup**: Finding all ClusterPermissions for the user/group
3. **Binding Matching**: Matching ClusterPermissions to discoverable ClusterRoles
4. **Aggregation**: Combining bindings from multiple ClusterPermissions
5. **Response Generation**: Creating UserPermission objects with bindings and ClusterRole definitions

**GET Operation:**

1. **ClusterRole Validation**: Checking if the requested ClusterRole is discoverable
2. **Permission Check**: Verifying user has bindings to the ClusterRole
3. **Binding Aggregation**: Collecting all bindings for the specific ClusterRole
4. **Response Generation**: Creating UserPermission object with complete details

### Commands

#### `setup` - Create Test Data

Creates discoverable ClusterRoles and ClusterPermissions:

```bash
KUBECONFIG=/path/to/config bash test-userpermission-performance.sh setup
```

This creates:

- Discoverable ClusterRoles with the required label
- Multiple ClusterPermission objects per cluster
- RBAC for test user/group
- Test namespace bindings across all clusters

#### `test` - Run Performance Test

Measures API response times for both LIST and GET operations:

```bash
KUBECONFIG=/path/to/config NUM_ITERATIONS=100 bash test-userpermission-performance.sh test
```

Output includes:

- LIST operation: Response time, ClusterRoles returned
- GET operation: Response time, success rate
- Min/max/average response times
- Throughput (requests/second)

#### `full` - Complete Test Suite

Runs setup → test → cleanup:

```bash
KUBECONFIG=/path/to/config bash test-userpermission-performance.sh full
```

#### `cleanup` - Remove Test Data

Removes all test ClusterRoles and ClusterPermissions:

```bash
KUBECONFIG=/path/to/config bash test-userpermission-performance.sh cleanup
```

### Test Scenarios

#### Scenario 1: Small Scale

Quick test with minimal resources:

```bash
NUM_CLUSTERS=2 \
NAMESPACES_PER_CLUSTER=10 \
NUM_CLUSTERPERMISSIONS=3 \
NUM_CLUSTERROLES=3 \
NUM_ITERATIONS=20 \
bash test-userpermission-performance.sh full
```

**Result:** 3 ClusterRoles, 6 ClusterPermissions, 20 namespace bindings

#### Scenario 2: Medium Scale

Realistic production-like test:

```bash
NUM_CLUSTERS=10 \
NAMESPACES_PER_CLUSTER=20 \
NUM_CLUSTERPERMISSIONS=5 \
NUM_CLUSTERROLES=10 \
NUM_ITERATIONS=50 \
bash test-userpermission-performance.sh full
```

**Result:** 10 ClusterRoles, 50 ClusterPermissions, 200 namespace bindings

#### Scenario 3: Large Scale

Stress test with many ClusterRoles:

```bash
NUM_CLUSTERS=20 \
NAMESPACES_PER_CLUSTER=50 \
NUM_CLUSTERPERMISSIONS=10 \
NUM_CLUSTERROLES=20 \
NUM_ITERATIONS=100 \
bash test-userpermission-performance.sh full
```

**Result:** 20 ClusterRoles, 200 ClusterPermissions, 1000 namespace bindings

#### Scenario 4: High ClusterPermission Density

Test with many ClusterPermissions per cluster:

```bash
NUM_CLUSTERS=5 \
NAMESPACES_PER_CLUSTER=100 \
NUM_CLUSTERPERMISSIONS=20 \
NUM_CLUSTERROLES=15 \
NUM_ITERATIONS=50 \
bash test-userpermission-performance.sh full
```

**Result:** 15 ClusterRoles, 100 ClusterPermissions, 500 namespace bindings

### Sample Output

```
[INFO] Creating 5 discoverable ClusterRoles...
[INFO]   Created 5 discoverable ClusterRoles
[INFO] Creating 5 ClusterPermissions for cluster-1 (50 namespaces total)...
[INFO]   Created 5 ClusterPermissions for cluster-1
[INFO] Setup complete!
[INFO] Total discoverable ClusterRoles: 5
[INFO] Total ClusterPermissions created: 50
[INFO] Total namespace bindings: 500

[INFO] Measuring LIST API performance with 50 iterations...

Iteration | Response Time (ms) | ClusterRoles Returned
----------|-------------------|----------------------
        1 |               156 |                    5
        2 |               142 |                    5
        3 |               138 |                    5
...

[INFO] LIST Performance Summary:
  Average response time: 145 ms
  Min response time: 132 ms
  Max response time: 178 ms
  ClusterRoles returned: 5
  Throughput: 6.90 requests/second

==========================================

[INFO] Measuring GET API performance with 50 iterations...

Iteration | Response Time (ms) | Success
----------|-------------------|--------
        1 |                89 |     Yes
        2 |                76 |     Yes
        3 |                82 |     Yes
...

[INFO] GET Performance Summary:
  Average response time: 83 ms
  Min response time: 71 ms
  Max response time: 102 ms
  Successful requests: 50/50
  Throughput: 12.05 requests/second
```

### Performance Recommendations

#### Development Testing

```bash
NUM_CLUSTERS=2 NUM_CLUSTERPERMISSIONS=3 NAMESPACES_PER_CLUSTER=10 NUM_CLUSTERROLES=3
```

**3 ClusterRoles, 6 ClusterPermissions, 20 bindings** - Quick feedback

#### Realistic Testing

```bash
NUM_CLUSTERS=10 NUM_CLUSTERPERMISSIONS=5 NAMESPACES_PER_CLUSTER=20 NUM_CLUSTERROLES=10
```

**10 ClusterRoles, 50 ClusterPermissions, 200 bindings** - Realistic scale

#### Performance Benchmarking

```bash
NUM_CLUSTERS=20 NUM_CLUSTERPERMISSIONS=10 NAMESPACES_PER_CLUSTER=50 NUM_CLUSTERROLES=20
```

**20 ClusterRoles, 200 ClusterPermissions, 1000 bindings** - Stress test

#### Cache Performance Test

```bash
NUM_CLUSTERS=10 NUM_CLUSTERPERMISSIONS=20 NAMESPACES_PER_CLUSTER=50 NUM_CLUSTERROLES=30 NUM_ITERATIONS=200
```

**30 ClusterRoles, 200 ClusterPermissions** - Tests caching efficiency

### Expected Performance

Baseline expectations (depends on cluster resources and caching):

| Scale | ClusterRoles | ClusterPermissions | Bindings | LIST Response | GET Response |
|-------|--------------|-------------------|----------|---------------|--------------|
| Small | 5 | 25 | 100 | < 100ms | < 50ms |
| Medium | 10 | 50 | 500 | < 200ms | < 100ms |
| Large | 20 | 200 | 2000 | < 500ms | < 200ms |
| Very Large | 50 | 500 | 5000 | < 1s | < 500ms |

**Good Performance Indicators:**

- GET operations faster than LIST operations (due to targeted lookup)
- Response time scales linearly with number of ClusterRoles
- Consistent performance across iterations (caching working)
- Min/max times within 2x of average
- No timeout errors
- High success rate (100%) for GET operations

### Troubleshooting

#### API Not Found

Check if the API server is running and discoverable ClusterRoles exist:

```bash
kubectl get clusterroles -l clusterview.open-cluster-management.io/discoverable=true
```

#### Permission Errors

Verify RBAC is created:

```bash
kubectl get clusterrolebinding test-user-userpermissions
kubectl get clusterrole userpermissions-viewer -o yaml
```

#### Slow Performance

1. Check API server resources:

   ```bash
   kubectl top pods -n kube-system
   ```

2. Review aggregated API server logs:

   ```bash
   kubectl logs -n open-cluster-management -l app=cluster-proxy-addon-user --tail=100
   ```

3. Reduce test scale:

   ```bash
   NUM_CLUSTERS=3 NUM_CLUSTERROLES=5 bash test-userpermission-performance.sh setup
   ```

#### Cache Issues

If performance degrades over iterations:

1. Check for memory issues in the aggregated API server
2. Verify cache is being used (performance should improve after first request)
3. Review cache implementation and TTL settings

### Performance Optimization Tips

Based on test results, consider:

1. **Caching**: Ensure UserPermission cache is working efficiently
2. **Indexing**: Optimize ClusterPermission indexing by subject
3. **ClusterRole Filtering**: Only process discoverable ClusterRoles
4. **Lazy Loading**: Load ClusterRole definitions only when needed
5. **Batch Processing**: Process multiple ClusterPermissions concurrently
6. **Response Size**: Consider pagination for users with many ClusterRoles

### Next Steps

After gathering performance data:

1. Compare LIST vs GET operation performance
2. Test with different ClusterRole sizes (number of rules)
3. Identify bottlenecks (discovery vs lookup vs aggregation)
4. Profile the aggregated API server if needed
5. Test cache hit rates and efficiency
6. Consider implementing optimizations listed above

### References

- API Implementation: [pkg/proxyserver/rest/userpermission/userpermission.go](../../pkg/proxyserver/rest/userpermission/userpermission.go)
- UserPermission Cache: [pkg/cache/userpermission.go](../../pkg/cache/userpermission.go)
- API Types: [pkg/proxyserver/apis/clusterview/v1alpha1/types_userpermission.go](../../pkg/proxyserver/apis/clusterview/v1alpha1/types_userpermission.go)

---

## Mock Cluster Setup

Script: `setup-mock-clusters.sh`

### Overview

The mock cluster setup script creates ManagedCluster resources and their corresponding namespaces without deploying actual Kubernetes clusters. This is ideal for performance testing, development, and CI/CD environments.

**Key Features:**

- Creates lightweight mock ManagedCluster CRs
- Generates cluster namespaces with proper labels
- No Docker or additional infrastructure required
- Fast setup and teardown (seconds)
- Compatible with all performance test scripts

### Quick Start

```bash
# Create 10 mock clusters
./setup-mock-clusters.sh setup

# Run performance tests
./test-kubevirtprojects-performance.sh full
./test-userpermission-performance.sh full

# Cleanup
./setup-mock-clusters.sh cleanup
```

### Commands

#### `setup` - Create Mock Clusters

Creates ManagedCluster resources and namespaces:

```bash
./setup-mock-clusters.sh setup
```

This creates:

- ManagedCluster CRs with minimal required fields
- Cluster namespaces with `open-cluster-management.io/cluster-name` label
- Mock cluster status (Available, Joined)
- Performance test labels for easy identification

#### `list` - List Mock Clusters

View all created mock clusters:

```bash
./setup-mock-clusters.sh list
```

#### `cleanup` - Remove Mock Clusters

Delete all mock clusters and their namespaces:

```bash
./setup-mock-clusters.sh cleanup
```

### Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NUM_CLUSTERS` | Number of mock clusters to create | 10 |
| `CLUSTER_PREFIX` | Prefix for cluster names | perf-test-cluster |
| `KUBECONFIG` | Path to kubeconfig file | ~/.kube/config |

### Usage Examples

#### Basic Usage

```bash
# Default setup (10 clusters)
./setup-mock-clusters.sh setup
```

#### Custom Number of Clusters

```bash
# Create 20 mock clusters
NUM_CLUSTERS=20 ./setup-mock-clusters.sh setup
```

#### Custom Cluster Prefix

```bash
# Use custom naming
CLUSTER_PREFIX=my-test-cluster NUM_CLUSTERS=5 ./setup-mock-clusters.sh setup
```

#### Full Workflow

```bash
# 1. Create mock clusters
NUM_CLUSTERS=15 ./setup-mock-clusters.sh setup

# 2. List to verify
./setup-mock-clusters.sh list

# 3. Run performance tests
NUM_CLUSTERS=15 ./test-kubevirtprojects-performance.sh full

# 4. Cleanup
./setup-mock-clusters.sh cleanup
```

### What Gets Created

For each mock cluster, the script creates:

**1. Namespace:**

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: perf-test-cluster-1
  labels:
    open-cluster-management.io/cluster-name: perf-test-cluster-1
```

**2. ManagedCluster:**

```yaml
apiVersion: cluster.open-cluster-management.io/v1
kind: ManagedCluster
metadata:
  name: perf-test-cluster-1
  labels:
    test: performance
    performance-test: "true"
spec:
  hubAcceptsClient: true
  managedClusterClientConfigs:
  - url: https://test-cluster-1.example.com:6443
status:
  conditions:
  - type: ManagedClusterConditionAvailable
    status: "True"
  - type: ManagedClusterJoined
    status: "True"
```

### Integration with Performance Tests

Both performance test scripts automatically work with mock clusters:

```bash
# Setup mock clusters
NUM_CLUSTERS=10 ./setup-mock-clusters.sh setup

# Run kubevirtprojects test (will use all 10 clusters)
NUM_CLUSTERS=10 ./test-kubevirtprojects-performance.sh full

# Run userpermissions test (will use all 10 clusters)
NUM_CLUSTERS=10 ./test-userpermission-performance.sh full

# Cleanup
./setup-mock-clusters.sh cleanup
```

### Prerequisites

The script requires:

1. **kubectl** - Configured and connected to your cluster
2. **ManagedCluster CRD** - Must be installed (comes with Open Cluster Management)

The script automatically validates these prerequisites before running.

### Troubleshooting

#### CRD Not Found

If you see "ManagedCluster CRD not found":

```bash
# Verify OCM is installed
kubectl get crd managedclusters.cluster.open-cluster-management.io
```

Install Open Cluster Management if needed.

#### Permission Errors

Ensure you have cluster-admin permissions:

```bash
kubectl auth can-i create managedclusters
kubectl auth can-i create namespaces
```

#### Clusters Already Exist

The script skips existing clusters automatically:

```bash
# First run - creates clusters
./setup-mock-clusters.sh setup

# Second run - skips existing, creates only new ones
NUM_CLUSTERS=15 ./setup-mock-clusters.sh setup
```

### Cleanup Behavior

The cleanup command is safe and targeted:

- Only removes clusters matching the configured prefix
- Only removes clusters with `test: performance` label
- Does not affect production managed clusters
- Handles stuck namespaces gracefully

### Best Practices

**For Development:**

```bash
NUM_CLUSTERS=5 ./setup-mock-clusters.sh setup
```

Small number for quick testing.

**For Performance Testing:**

```bash
NUM_CLUSTERS=20 ./setup-mock-clusters.sh setup
```

Realistic scale for benchmarking.

**For CI/CD:**

```bash
# In your CI pipeline
NUM_CLUSTERS=10 ./setup-mock-clusters.sh setup
./test-kubevirtprojects-performance.sh test
./test-userpermission-performance.sh test
./setup-mock-clusters.sh cleanup
```

Always cleanup in CI to avoid resource accumulation.

### Comparison: Mock vs Real Clusters

| Aspect | Mock Clusters | Real Clusters |
|--------|--------------|---------------|
| Setup Time | < 10 seconds | 5-10 minutes |
| Resources | Minimal (CRs only) | High (Docker, kind, etc.) |
| Use Case | Development, Testing | Production, E2E Testing |
| Cleanup | Instant | 1-2 minutes |
| Realism | Limited | Full |
| Prerequisites | kubectl only | Docker, kind, etc. |

**Recommendation:** Use mock clusters for performance testing unless you specifically need to test real cluster connectivity.

### Script Output Example

```bash
$ NUM_CLUSTERS=3 ./setup-mock-clusters.sh setup
[INFO] Creating 3 mock ManagedCluster resources...
[DEBUG] Creating mock ManagedCluster: perf-test-cluster-1
[DEBUG] Creating mock ManagedCluster: perf-test-cluster-2
[DEBUG] Creating mock ManagedCluster: perf-test-cluster-3
[INFO] Setup complete!
[INFO] Created: 3 new ManagedClusters
[INFO] Total ManagedClusters: 3

[INFO] Cluster namespaces created:
perf-test-cluster-1   Active   5s
perf-test-cluster-2   Active   4s
perf-test-cluster-3   Active   3s
```
