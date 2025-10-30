# KubeVirt Projects API Performance Testing

Performance testing tool for the `kubevirtprojects.clusterview.open-cluster-management.io` API.

## Overview

The `kubevirtprojects` API aggregates project access information across multiple managed clusters based on ClusterPermission objects. This script helps test performance under various conditions.

**Key Feature:** Creates **multiple ClusterPermission objects** per cluster namespace to better simulate real-world scenarios and test indexer performance.

## Prerequisites

1. **kubectl** - Configured and connected to your test cluster
2. **jq** - JSON processor for parsing API responses
3. **bc** - Calculator for performance metrics
4. **ClusterPermission CRD** - Installed in the cluster
5. **kubevirtprojects API** - Available and accessible

### Installing Prerequisites

```bash
# macOS
brew install jq bc

# Linux (Ubuntu/Debian)
sudo apt-get install jq bc

# Linux (RHEL/CentOS)
sudo yum install jq bc
```

## Quick Start

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

## Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NUM_CLUSTERS` | Number of cluster namespaces to use | 10 |
| `PROJECTS_PER_CLUSTER` | Total projects per cluster | 10 |
| `NUM_CLUSTERPERMISSIONS` | ClusterPermission objects per cluster | 5 |
| `NUM_ITERATIONS` | Performance test iterations | 10 |
| `KUBECONFIG` | Path to kubeconfig file | ~/.kube/config |

## How It Works

### Multiple ClusterPermissions Distribution

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

### What Gets Tested

The API performance for:
1. **Index lookup**: Finding all ClusterPermissions for the user/group
2. **Iteration**: Processing each ClusterPermission object
3. **Extraction**: Getting roleBindings from each ClusterPermission
4. **Aggregation**: Combining projects from multiple ClusterPermissions
5. **Deduplication**: Removing duplicate projects (using sets)
6. **Sorting**: Sorting results by cluster name
7. **Generation**: Creating PartialObjectMetadata for each project

## Commands

### `setup` - Create Test Data

Creates ClusterPermissions in existing cluster namespaces:

```bash
KUBECONFIG=/path/to/config bash test-kubevirtprojects-performance.sh setup
```

This creates:
- Multiple ClusterPermission objects per cluster
- RBAC for test user/group
- Test projects across all clusters

### `test` - Run Performance Test

Measures API response times:

```bash
KUBECONFIG=/path/to/config NUM_ITERATIONS=100 bash test-kubevirtprojects-performance.sh test
```

Output includes:
- Response time per iteration
- Min/max/average response times
- Number of projects returned
- Throughput (requests/second)

### `full` - Complete Test Suite

Runs setup → test → cleanup:

```bash
KUBECONFIG=/path/to/config bash test-kubevirtprojects-performance.sh full
```

### `cleanup` - Remove Test Data

Removes all test ClusterPermissions:

```bash
KUBECONFIG=/path/to/config bash test-kubevirtprojects-performance.sh cleanup
```

## Test Scenarios

### Scenario 1: Indexer Stress Test

Test with many ClusterPermission objects:

```bash
NUM_CLUSTERS=5 \
PROJECTS_PER_CLUSTER=100 \
NUM_CLUSTERPERMISSIONS=20 \
NUM_ITERATIONS=50 \
bash test-kubevirtprojects-performance.sh full
```

**Result:** 100 ClusterPermissions (5 clusters × 20 CPs each), 500 total projects

### Scenario 2: Large ClusterPermissions

Test with fewer but larger ClusterPermission objects:

```bash
NUM_CLUSTERS=10 \
PROJECTS_PER_CLUSTER=100 \
NUM_CLUSTERPERMISSIONS=2 \
NUM_ITERATIONS=50 \
bash test-kubevirtprojects-performance.sh full
```

**Result:** 20 ClusterPermissions (10 × 2), each with 50 projects

### Scenario 3: Balanced Distribution

Moderate and balanced test:

```bash
NUM_CLUSTERS=10 \
PROJECTS_PER_CLUSTER=50 \
NUM_CLUSTERPERMISSIONS=5 \
NUM_ITERATIONS=100 \
bash test-kubevirtprojects-performance.sh full
```

**Result:** 50 ClusterPermissions, 500 total projects

### Scenario 4: High ClusterPermission Count

Maximum ClusterPermission objects:

```bash
NUM_CLUSTERS=3 \
PROJECTS_PER_CLUSTER=60 \
NUM_CLUSTERPERMISSIONS=30 \
NUM_ITERATIONS=50 \
bash test-kubevirtprojects-performance.sh full
```

**Result:** 90 ClusterPermissions (3 × 30), 2 projects per CP

## Sample Output

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

## Performance Recommendations

### Development Testing
```bash
NUM_CLUSTERS=2 NUM_CLUSTERPERMISSIONS=3 PROJECTS_PER_CLUSTER=10
```
**6 ClusterPermissions, 20 projects** - Quick feedback

### Realistic Testing
```bash
NUM_CLUSTERS=10 NUM_CLUSTERPERMISSIONS=5 PROJECTS_PER_CLUSTER=50
```
**50 ClusterPermissions, 500 projects** - Realistic scale

### Performance Benchmarking
```bash
NUM_CLUSTERS=20 NUM_CLUSTERPERMISSIONS=10 PROJECTS_PER_CLUSTER=100
```
**200 ClusterPermissions, 2000 projects** - Stress test

### Indexer Stress Test
```bash
NUM_CLUSTERS=10 NUM_CLUSTERPERMISSIONS=50 PROJECTS_PER_CLUSTER=100
```
**500 ClusterPermissions** - Tests indexer specifically

## Expected Performance

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

## Troubleshooting

### API Not Found

Check if the API server is running and CRD is installed:

```bash
kubectl get crd clusterpermissions.rbac.open-cluster-management.io
```

### Permission Errors

Verify RBAC is created:

```bash
kubectl get clusterrolebinding test-user-kubevirtprojects
kubectl get clusterrole kubevirtprojects-viewer -o yaml
```

### Slow Performance

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

### Out of Memory

- Reduce `NUM_CLUSTERS` or `NUM_CLUSTERPERMISSIONS`
- Check for memory leaks in the implementation
- Increase API server memory limits

## Performance Optimization Tips

Based on test results, consider:

1. **Caching**: Implement caching for ClusterPermission lookups
2. **Indexing**: Ensure proper indexing on subject names/groups
3. **Pagination**: Add support for pagination with large result sets
4. **Filtering**: Allow filtering by cluster to reduce result size
5. **Concurrent Processing**: Process multiple clusters concurrently
6. **Watch API**: Use watch API instead of repeated polls

## Next Steps

After gathering performance data:

1. Compare results with different ClusterPermission distributions
2. Identify bottlenecks (indexing vs iteration vs aggregation)
3. Profile the API server if needed
4. Consider implementing optimizations listed above

## References

- API Implementation: [pkg/proxyserver/rest/project/rest.go](../../pkg/proxyserver/rest/project/rest.go)
- Project Listing: [pkg/proxyserver/rest/project/list.go](../../pkg/proxyserver/rest/project/list.go)
- ClusterPermission Indexing: [pkg/proxyserver/rest/project/index.go](../../pkg/proxyserver/rest/project/index.go)
