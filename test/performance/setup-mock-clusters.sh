#!/bin/bash

# Mock ManagedCluster Setup Script for Performance Testing
# This script creates mock ManagedCluster resources and their namespaces
# without creating actual Kubernetes clusters.

set -e

# Configuration
NUM_CLUSTERS=${NUM_CLUSTERS:-10}
CLUSTER_PREFIX=${CLUSTER_PREFIX:-perf-test-cluster}
KUBECONFIG=${KUBECONFIG:-~/.kube/config}

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# Unset proxy environment variables per configuration
unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy

# Create a mock ManagedCluster resource
create_mock_cluster() {
    local cluster_name=$1
    local cluster_index=$2

    log_debug "Creating mock ManagedCluster: ${cluster_name}"

    # Create the cluster namespace first
    cat <<EOF | kubectl apply -f - >/dev/null
apiVersion: v1
kind: Namespace
metadata:
  name: ${cluster_name}
  labels:
    open-cluster-management.io/cluster-name: ${cluster_name}
EOF

    # Create the ManagedCluster resource
    cat <<EOF | kubectl apply -f - >/dev/null
apiVersion: cluster.open-cluster-management.io/v1
kind: ManagedCluster
metadata:
  name: ${cluster_name}
  labels:
    cluster.open-cluster-management.io/clusterset: default
    test: performance
    performance-test: "true"
spec:
  hubAcceptsClient: true
  leaseDurationSeconds: 60
  managedClusterClientConfigs:
  - caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM2akNDQWRLZ0F3SUJBZ0lCQVRBTkJna3Foa2lHOXcwQkFRc0ZBREFtTVNRd0lnWURWUVFEREJ0dmNHVnUKTFhOb2FXWjBMWE5wWjI1bGNrQXhOakUyTVRBd01qQTBNQjRYRFRJeE1EWXdOVEF5TURBd05Gb1hEVE14TURZdwpNekF5TURBd05Gb3dKakVrTUNJR0ExVUVBd3diYjNCbGJpMXphR2xtZEMxemFXZHVaWEpBTVRZeE5qRXdNREV3Ck5EQ0NBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCQU1Va1dQV2R5aE9wbCthU0ZZaDUKWmI1QnAvL0wrZkhjR3JMY0d6VUJnSE1vYWQ5Nng1SXBBRXZ4VXBIM3lRaWJyZGJ6WHB4TVVudWphc2ZMYWd5QQpNb0hXcGhGL0Z1V0xTVXEvRU9xQXRXdVN2Zmg4cDdHalphWmVZc3RVWHVvQnZWcjlUaStCdFo2blVZNmpBVFc5CnNMWEdyL1BWN01Hc3FmZCtoN2pGYmNERzg1OEVnY0s4Z25aQk1MMGdNTWxod0orU3ptOVB5QTBicVNPWlQ2SXcKNjB0emFUeWNyVUJnc0F3VkR3SmhKeWQ1QSsyOHhDbFkrTjRQc1daa2xFeTJSTVcxNDhtT0dTTXBGaDJ2azRFWApJZllrTzVGNkM1SkxTQzl4QkZzSlp3L3lyY0ViTlozSmtLeUJUYi9NbGlpZVdGcFcxTmNteWJSeWNIZVlxWEFzCk8wc0NBd0VBQWFOQ01FQXdEZ1lEVlIwUEFRSC9CQVFEQWdLa01BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0hRWUQKVlIwT0JCWUVGQTgyMzEvWGh4L3VDZkJ5akJPTitWRjUwK3J2TUEwR0NTcUdTSWIzRFFFQkN3VUFBNElCQVFDVQpCWUFzb3BQWUlnRzJZd3Y2MVVmV2Z3NVc2Z0JZQXhZNTdJZVZCb3FCVm9MV3FFOHB2TVhCdGY0TTk2dDl6TjFUCnE3NThJT0lxRU9VQVJNVzNQNHRZS21nWS9wRjJhaXJHMFcwemxEUWlIWUdDRUNxQVRCWmFBZ0NLME9jQWtBT2EKWW16NkdVYmhxMThsdG5LcmlOeStLcHFkZ09TcGJQRTJvSTJDWG9KTEdvNGMxUHZ0MlJLSEdNdUVzSGdHQmRISApSSTVZZFovb2prYjlCdCtybWRPeGR4MXZDd3NmdlREZGwzTjM0Q1Y0bXRKSFJvTytPb0U3cE5mT1R4VG1MSHRPCk92SDhCRFhJS2c3ZUIvVWN4QS9tRWxGWGZ2VDRiQzhnWVBJUlFWTFlOYlpkRGNQazJTWklxZWlhU2g5Nm5oaTgKelJlbzZ5RzQrdFFKM2RrdnlnTWEKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQ==
    url: https://test-cluster-${cluster_index}.example.com:6443
status:
  allocatable:
    cpu: "4"
    memory: 16Gi
  capacity:
    cpu: "4"
    memory: 16Gi
  clusterClaims:
  - name: id.k8s.io
    value: perf-test-${cluster_index}
  - name: platform.open-cluster-management.io
    value: Mock
  - name: product.open-cluster-management.io
    value: Kubernetes
  conditions:
  - lastTransitionTime: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    message: Accepted by hub cluster admin
    reason: HubClusterAdminAccepted
    status: "True"
    type: HubAcceptedManagedCluster
  - lastTransitionTime: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    message: Managed cluster joined
    reason: ManagedClusterJoined
    status: "True"
    type: ManagedClusterJoined
  - lastTransitionTime: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    message: Managed cluster is available
    reason: ManagedClusterAvailable
    status: "True"
    type: ManagedClusterConditionAvailable
  version:
    kubernetes: v1.30.0
EOF
}

# Setup mock clusters
setup_mock_clusters() {
    log_info "Creating ${NUM_CLUSTERS} mock ManagedCluster resources..."

    local created=0
    local skipped=0

    for ((i=1; i<=NUM_CLUSTERS; i++)); do
        local cluster_name="${CLUSTER_PREFIX}-${i}"

        # Check if cluster already exists
        if kubectl get managedcluster "${cluster_name}" >/dev/null 2>&1; then
            log_warn "ManagedCluster ${cluster_name} already exists, skipping..."
            skipped=$((skipped + 1))
            continue
        fi

        create_mock_cluster "${cluster_name}" "${i}"
        created=$((created + 1))
    done

    log_info "Setup complete!"
    log_info "Created: ${created} new ManagedClusters"
    if [ $skipped -gt 0 ]; then
        log_info "Skipped: ${skipped} existing ManagedClusters"
    fi
    log_info "Total ManagedClusters: ${NUM_CLUSTERS}"

    echo ""
    log_info "Cluster namespaces created:"
    kubectl get namespaces -l open-cluster-management.io/cluster-name \
        | grep "${CLUSTER_PREFIX}" || true
}

# List mock clusters
list_mock_clusters() {
    log_info "Mock ManagedClusters for performance testing:"
    echo ""
    kubectl get managedclusters -l test=performance --no-headers 2>/dev/null || \
        echo "No mock clusters found"

    echo ""
    log_info "Cluster namespaces:"
    kubectl get namespaces -l open-cluster-management.io/cluster-name \
        --no-headers 2>/dev/null | grep "${CLUSTER_PREFIX}" || \
        echo "No cluster namespaces found"
}

# Cleanup mock clusters
cleanup_mock_clusters() {
    log_warn "Cleaning up mock ManagedClusters..."

    local deleted_clusters=0
    local deleted_namespaces=0

    # Delete ManagedClusters with performance test label
    local clusters=$(kubectl get managedclusters -l test=performance \
        -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

    if [ -n "$clusters" ]; then
        for cluster in $clusters; do
            if [[ $cluster == ${CLUSTER_PREFIX}-* ]]; then
                log_debug "Deleting ManagedCluster: ${cluster}"
                kubectl delete managedcluster "${cluster}" \
                    --ignore-not-found=true >/dev/null 2>&1 || true
                deleted_clusters=$((deleted_clusters + 1))
            fi
        done
    fi

    # Delete cluster namespaces
    local namespaces=$(kubectl get namespaces \
        -l open-cluster-management.io/cluster-name \
        -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

    if [ -n "$namespaces" ]; then
        for ns in $namespaces; do
            if [[ $ns == ${CLUSTER_PREFIX}-* ]]; then
                log_debug "Deleting namespace: ${ns}"
                kubectl delete namespace "${ns}" \
                    --ignore-not-found=true >/dev/null 2>&1 || true
                deleted_namespaces=$((deleted_namespaces + 1))
            fi
        done
    fi

    log_info "Cleanup complete!"
    log_info "Deleted ${deleted_clusters} ManagedClusters"
    log_info "Deleted ${deleted_namespaces} namespaces"
}

# Validate prerequisites
check_prerequisites() {
    local missing=()

    command -v kubectl >/dev/null 2>&1 || missing+=("kubectl")

    if [ ${#missing[@]} -gt 0 ]; then
        echo "ERROR: Missing required tools: ${missing[*]}"
        echo "Please install them before running this script"
        exit 1
    fi

    # Check if ManagedCluster CRD exists
    if ! kubectl get crd managedclusters.cluster.open-cluster-management.io \
        >/dev/null 2>&1; then
        echo "ERROR: ManagedCluster CRD not found"
        echo "Please install Open Cluster Management before running this script"
        exit 1
    fi
}

# Main
case "${1:-setup}" in
    setup)
        check_prerequisites
        setup_mock_clusters
        ;;
    list)
        list_mock_clusters
        ;;
    cleanup)
        cleanup_mock_clusters
        ;;
    *)
        cat <<EOF
Mock ManagedCluster Setup for Performance Testing

This script creates mock ManagedCluster resources for performance testing
without creating actual Kubernetes clusters.

Usage: $0 <command>

Commands:
  setup   - Create mock ManagedClusters and namespaces (default)
  list    - List existing mock ManagedClusters
  cleanup - Remove all mock ManagedClusters and namespaces

Environment Variables:
  NUM_CLUSTERS     - Number of mock clusters to create (default: 10)
  CLUSTER_PREFIX   - Prefix for cluster names (default: perf-test-cluster)
  KUBECONFIG       - Path to kubeconfig (default: ~/.kube/config)

Examples:
  # Create 10 mock clusters (default)
  $0 setup

  # Create 20 mock clusters
  NUM_CLUSTERS=20 $0 setup

  # Use custom prefix
  CLUSTER_PREFIX=my-test-cluster NUM_CLUSTERS=5 $0 setup

  # List all mock clusters
  $0 list

  # Cleanup all mock clusters
  $0 cleanup

  # Full workflow
  $0 setup
  $0 list
  # Run performance tests...
  $0 cleanup

Notes:
  - Mock clusters are created with minimal required fields
  - Each cluster gets a namespace with the managedCluster label
  - Clusters are labeled with 'test: performance' for easy identification
  - Cleanup removes only clusters matching the prefix pattern
EOF
        ;;
esac
