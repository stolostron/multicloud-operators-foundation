#!/bin/bash

# Quick performance test script for kubevirtprojects API
# This is a simplified version that avoids namespace termination issues

set -e

# Configuration
NUM_CLUSTERS=${NUM_CLUSTERS:-10}
PROJECTS_PER_CLUSTER=${PROJECTS_PER_CLUSTER:-10}
NUM_CLUSTERPERMISSIONS=${NUM_CLUSTERPERMISSIONS:-5}  # Number of ClusterPermission objects per cluster
NUM_ITERATIONS=${NUM_ITERATIONS:-10}
KUBECONFIG=${KUBECONFIG:-~/.kube/config}

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Create multiple ClusterPermissions with projects for existing cluster
create_test_clusterpermissions() {
    local cluster_name=$1
    local num_projects=$2
    local num_clusterpermissions=${3:-5}  # Default: create 5 ClusterPermissions per cluster

    log_info "Creating ${num_clusterpermissions} ClusterPermissions for ${cluster_name} (${num_projects} projects total)..."

    # Calculate how many projects per ClusterPermission
    local projects_per_cp=$((num_projects / num_clusterpermissions))
    if [ $projects_per_cp -lt 1 ]; then
        projects_per_cp=1
    fi

    # Create multiple ClusterPermission objects
    for ((cp_idx=1; cp_idx<=num_clusterpermissions; cp_idx++)); do
        local cp_name="perf-test-cp-${cp_idx}"

        # Calculate project range for this ClusterPermission
        local start_project=$(( (cp_idx - 1) * projects_per_cp + 1 ))
        local end_project=$((cp_idx * projects_per_cp))

        # For the last ClusterPermission, include any remaining projects
        if [ $cp_idx -eq $num_clusterpermissions ]; then
            end_project=$num_projects
        fi

        local role_bindings=""
        for ((i=start_project; i<=end_project; i++)); do
            local project_name="project-${i}"
            role_bindings="${role_bindings}
  - namespace: ${project_name}
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: kubevirt.io:edit
    subject:
      apiGroup: rbac.authorization.k8s.io
      kind: Group
      name: test-group"
        done

        # Create the ClusterPermission
        cat <<EOF | kubectl apply -f -
apiVersion: rbac.open-cluster-management.io/v1alpha1
kind: ClusterPermission
metadata:
  name: ${cp_name}
  namespace: ${cluster_name}
spec:
  roleBindings:${role_bindings}
EOF
    done

    log_info "  Created ${num_clusterpermissions} ClusterPermissions for ${cluster_name}"
}

# Setup test on existing clusters
setup_on_existing_clusters() {
    log_info "Finding existing managed cluster namespaces..."

    # Get list of managed cluster namespaces
    local clusters=$(kubectl get namespaces -l cluster.open-cluster-management.io/managedCluster -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

    if [ -z "$clusters" ]; then
        log_warn "No managed cluster namespaces found"
        log_warn "Looking for any namespaces that look like clusters..."
        clusters=$(kubectl get namespaces -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | grep -E '^(cluster|local-cluster)' | head -${NUM_CLUSTERS} | tr '\n' ' ' || echo "")
    fi

    if [ -z "$clusters" ]; then
        echo "ERROR: No suitable cluster namespaces found"
        echo "Please ensure you have managed cluster namespaces or adjust the script"
        exit 1
    fi

    local cluster_array=($clusters)
    local cluster_count=${#cluster_array[@]}

    if [ $cluster_count -lt $NUM_CLUSTERS ]; then
        log_warn "Only found ${cluster_count} clusters, adjusting test to use available clusters"
        NUM_CLUSTERS=$cluster_count
    fi

    log_info "Using ${NUM_CLUSTERS} clusters for testing"
    echo "Clusters: ${cluster_array[@]:0:$NUM_CLUSTERS}"

    # Create test RBAC
    log_info "Creating test RBAC..."
    cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubevirtprojects-viewer
rules:
- apiGroups: ["clusterview.open-cluster-management.io"]
  resources: ["kubevirtprojects"]
  verbs: ["list", "get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: test-user-kubevirtprojects
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kubevirtprojects-viewer
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: test-user
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: test-group
EOF

    # Create ClusterPermissions in existing clusters
    for ((i=0; i<NUM_CLUSTERS; i++)); do
        local cluster=${cluster_array[$i]}
        create_test_clusterpermissions "$cluster" "$PROJECTS_PER_CLUSTER" "$NUM_CLUSTERPERMISSIONS"
    done

    log_info "Setup complete!"
    log_info "Total ClusterPermissions created: $((NUM_CLUSTERS * NUM_CLUSTERPERMISSIONS))"
    log_info "Total projects: $((NUM_CLUSTERS * PROJECTS_PER_CLUSTER))"
}

# Measure performance
measure_performance() {
    log_info "Measuring API performance with ${NUM_ITERATIONS} iterations..."

    local total_time=0
    local min_time=999999
    local max_time=0
    local project_count=0

    echo ""
    echo "Iteration | Response Time (ms) | Projects Returned"
    echo "----------|-------------------|------------------"

    for ((i=1; i<=NUM_ITERATIONS; i++)); do
        local start_time=$(date +%s%N)

        local output=$(kubectl get kubevirtprojects.clusterview.open-cluster-management.io \
            --as=test-user --as-group=test-group -o json 2>/dev/null || echo '{"items":[]}')

        local end_time=$(date +%s%N)
        local duration=$(( (end_time - start_time) / 1000000 ))

        if [ -n "$output" ]; then
            project_count=$(echo "$output" | jq '.items | length' 2>/dev/null || echo "0")
        fi

        printf "%9d | %17d | %17d\n" "$i" "$duration" "$project_count"

        total_time=$((total_time + duration))

        if [ $duration -lt $min_time ]; then
            min_time=$duration
        fi

        if [ $duration -gt $max_time ]; then
            max_time=$duration
        fi

        sleep 0.1
    done

    local avg_time=$((total_time / NUM_ITERATIONS))

    echo ""
    log_info "Performance Summary:"
    echo "  Average response time: ${avg_time} ms"
    echo "  Min response time: ${min_time} ms"
    echo "  Max response time: ${max_time} ms"
    echo "  Projects returned: ${project_count}"
    echo "  Throughput: $(echo "scale=2; 1000 / $avg_time" | bc 2>/dev/null || echo 'N/A') requests/second"
}

# Cleanup test ClusterPermissions
cleanup() {
    log_warn "Cleaning up test ClusterPermissions..."

    # Delete all ClusterPermissions matching the pattern
    local deleted=0
    for ((i=1; i<=20; i++)); do  # Support up to 20 ClusterPermissions per cluster
        local count=$(kubectl get clusterpermission "perf-test-cp-${i}" --all-namespaces --no-headers 2>/dev/null | wc -l | tr -d ' ')
        if [ "$count" -gt 0 ]; then
            kubectl delete clusterpermission "perf-test-cp-${i}" --all-namespaces --ignore-not-found=true 2>/dev/null || true
            deleted=$((deleted + count))
        fi
    done

    log_info "Deleted ${deleted} ClusterPermissions"

    log_info "Deleting test RBAC..."
    kubectl delete clusterrolebinding test-user-kubevirtprojects --ignore-not-found=true 2>/dev/null || true
    kubectl delete clusterrole kubevirtprojects-viewer --ignore-not-found=true 2>/dev/null || true

    log_info "Cleanup complete!"
}

# Main
case "${1:-test}" in
    setup)
        setup_on_existing_clusters
        ;;
    test)
        measure_performance
        ;;
    cleanup)
        cleanup
        ;;
    full)
        cleanup 2>/dev/null || true
        setup_on_existing_clusters
        echo ""
        measure_performance
        echo ""
        read -p "Keep test data? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            cleanup
        fi
        ;;
    *)
        cat <<EOF
Quick Performance Test for kubevirtprojects API

Usage: $0 <command>

Commands:
  setup   - Create test ClusterPermissions in existing cluster namespaces
  test    - Run performance test
  cleanup - Remove test ClusterPermissions
  full    - Run complete test (setup -> test -> cleanup)

Environment Variables:
  NUM_CLUSTERS            - Max number of clusters to use (default: 10)
  PROJECTS_PER_CLUSTER    - Projects per cluster (default: 10)
  NUM_CLUSTERPERMISSIONS  - Number of ClusterPermission objects per cluster (default: 5)
  NUM_ITERATIONS          - Test iterations (default: 10)
  KUBECONFIG              - Path to kubeconfig (default: ~/.kube/config)

Examples:
  # Quick test with defaults (10 clusters, 10 projects, 5 ClusterPermissions per cluster = 50 total CPs)
  $0 full

  # Custom scale with more ClusterPermissions
  NUM_CLUSTERS=5 PROJECTS_PER_CLUSTER=20 NUM_CLUSTERPERMISSIONS=10 NUM_ITERATIONS=50 $0 full

  # Stress test with many ClusterPermissions per cluster
  NUM_CLUSTERS=3 PROJECTS_PER_CLUSTER=50 NUM_CLUSTERPERMISSIONS=20 $0 full

  # Use specific kubeconfig
  KUBECONFIG=/path/to/kubeconfig $0 full
EOF
        ;;
esac
