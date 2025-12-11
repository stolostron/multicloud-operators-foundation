#!/bin/bash

# Quick performance test script for userpermissions API
# This is a simplified version that avoids namespace termination issues

set -e

# Configuration
NUM_CLUSTERS=${NUM_CLUSTERS:-10}
NAMESPACES_PER_CLUSTER=${NAMESPACES_PER_CLUSTER:-10}
NUM_CLUSTERPERMISSIONS=${NUM_CLUSTERPERMISSIONS:-5}  # Number of ClusterPermission objects per cluster
NUM_CLUSTERROLES=${NUM_CLUSTERROLES:-5}  # Number of discoverable ClusterRoles to create
NUM_ADMIN_BINDINGS_PER_CLUSTER=${NUM_ADMIN_BINDINGS_PER_CLUSTER:-3}  # Creates N admin ClusterRoles, N ClusterRoleBindings, N Roles, and N RoleBindings per cluster
NUM_VIEW_BINDINGS_PER_CLUSTER=${NUM_VIEW_BINDINGS_PER_CLUSTER:-2}  # Creates N view ClusterRoles, N ClusterRoleBindings, N Roles, and N RoleBindings per cluster
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

# Create discoverable ClusterRoles for testing
create_discoverable_clusterroles() {
    local num_clusterroles=$1

    log_info "Creating ${num_clusterroles} discoverable ClusterRoles..."

    for ((i=1; i<=num_clusterroles; i++)); do
        local cr_name="perf-test-clusterrole-${i}"

        cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ${cr_name}
  labels:
    clusterview.open-cluster-management.io/discoverable: "true"
    test: "performance"
rules:
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "create", "update", "delete"]
- apiGroups: [""]
  resources: ["pods", "services", "configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list"]
EOF
    done

    log_info "  Created ${num_clusterroles} discoverable ClusterRoles"
}

# Create multiple ClusterPermissions with bindings for existing cluster
create_test_clusterpermissions() {
    local cluster_name=$1
    local num_namespaces=$2
    local num_clusterpermissions=${3:-5}  # Default: create 5 ClusterPermissions per cluster

    log_info "Creating ${num_clusterpermissions} ClusterPermissions for ${cluster_name} (${num_namespaces} namespaces total)..."

    # Calculate how many namespaces per ClusterPermission
    local namespaces_per_cp=$((num_namespaces / num_clusterpermissions))
    if [ $namespaces_per_cp -lt 1 ]; then
        namespaces_per_cp=1
    fi

    # Create multiple ClusterPermission objects
    for ((cp_idx=1; cp_idx<=num_clusterpermissions; cp_idx++)); do
        local cp_name="perf-test-cp-${cp_idx}"

        # Calculate namespace range for this ClusterPermission
        local start_ns=$(( (cp_idx - 1) * namespaces_per_cp + 1 ))
        local end_ns=$((cp_idx * namespaces_per_cp))

        # For the last ClusterPermission, include any remaining namespaces
        if [ $cp_idx -eq $num_clusterpermissions ]; then
            end_ns=$num_namespaces
        fi

        # Determine which ClusterRole to reference (rotate through available ClusterRoles)
        local cr_idx=$(( (cp_idx - 1) % NUM_CLUSTERROLES + 1 ))
        local clusterrole_name="perf-test-clusterrole-${cr_idx}"

        local role_bindings=""
        local ns_idx
        for ((ns_idx=start_ns; ns_idx<=end_ns; ns_idx++)); do
            local namespace_name="namespace-${ns_idx}"
            role_bindings="${role_bindings}
  - namespace: ${namespace_name}
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: ${clusterrole_name}
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

# Create RBAC bindings that grant managedcluster admin/view permissions
# This tests the processor_admin_view.go code path
# For each cluster, creates multiple ClusterRoles, ClusterRoleBindings, Roles, and RoleBindings
create_admin_view_rbac() {
    local num_admin_bindings_per_cluster=${1:-3}  # Admin bindings per cluster
    local num_view_bindings_per_cluster=${2:-2}   # View bindings per cluster
    shift 2
    local cluster_array=("$@")    # Remaining args are cluster names

    log_info "Creating admin/view RBAC bindings for synthetic permissions..."
    log_info "  Per cluster: ${num_admin_bindings_per_cluster} admin ClusterRoles + ${num_admin_bindings_per_cluster} admin ClusterRoleBindings + ${num_admin_bindings_per_cluster} admin Roles/RoleBindings"
    log_info "  Per cluster: ${num_view_bindings_per_cluster} view ClusterRoles + ${num_view_bindings_per_cluster} view ClusterRoleBindings + ${num_view_bindings_per_cluster} view Roles/RoleBindings"

    # Process clusters in reverse order
    local num_clusters=${#cluster_array[@]}
    for ((cluster_idx=num_clusters-1; cluster_idx>=0; cluster_idx--)); do
        local cluster_name="${cluster_array[$cluster_idx]}"
        # 1. Create admin ClusterRoles and ClusterRoleBindings (1:1 mapping)
        for ((i=1; i<=num_admin_bindings_per_cluster; i++)); do
            local admin_cr_name="perf-test-admin-cr-${cluster_name}-${i}"
            local crb_name="perf-test-admin-crb-${cluster_name}-${i}"

            cat <<EOF | kubectl apply -f - 2>&1 || log_warn "Failed to create admin ClusterRole/ClusterRoleBinding for ${cluster_name}-${i}, continuing..."
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ${admin_cr_name}
  labels:
    test: "performance"
    cluster: "${cluster_name}"
    type: "admin"
rules:
- apiGroups: ["view.open-cluster-management.io"]
  resources: ["managedclusteractions"]
  verbs: ["create", "get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ${crb_name}
  labels:
    test: "performance"
    cluster: "${cluster_name}"
    type: "admin"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ${admin_cr_name}
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: test-user
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: test-group
EOF
        done

        # 2. Create admin Roles and RoleBindings for this cluster
        for ((i=1; i<=num_admin_bindings_per_cluster; i++)); do
            local admin_role_name="perf-test-admin-role-${cluster_name}-${i}"
            local admin_rb_name="perf-test-admin-rb-${cluster_name}-${i}"

            cat <<EOF | kubectl apply -f - 2>&1 || log_warn "Failed to create admin Role/RoleBinding for ${cluster_name}-${i}, continuing..."
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ${admin_role_name}
  namespace: ${cluster_name}
  labels:
    test: "performance"
    cluster: "${cluster_name}"
    type: "admin"
rules:
- apiGroups: ["view.open-cluster-management.io"]
  resources: ["managedclusteractions"]
  verbs: ["create", "get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ${admin_rb_name}
  namespace: ${cluster_name}
  labels:
    test: "performance"
    cluster: "${cluster_name}"
    type: "admin"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: ${admin_role_name}
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: test-user
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: test-group
EOF
        done

        # 3. Create view ClusterRoles and ClusterRoleBindings (1:1 mapping)
        for ((i=1; i<=num_view_bindings_per_cluster; i++)); do
            local view_cr_name="perf-test-view-cr-${cluster_name}-${i}"
            local crb_name="perf-test-view-crb-${cluster_name}-${i}"

            cat <<EOF | kubectl apply -f - 2>&1 || log_warn "Failed to create view ClusterRole/ClusterRoleBinding for ${cluster_name}-${i}, continuing..."
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ${view_cr_name}
  labels:
    test: "performance"
    cluster: "${cluster_name}"
    type: "view"
rules:
- apiGroups: ["view.open-cluster-management.io"]
  resources: ["managedclusterviews"]
  verbs: ["create", "get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ${crb_name}
  labels:
    test: "performance"
    cluster: "${cluster_name}"
    type: "view"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ${view_cr_name}
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: test-user
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: test-group
EOF
        done

        # 4. Create view Roles and RoleBindings for this cluster
        for ((i=1; i<=num_view_bindings_per_cluster; i++)); do
            local view_role_name="perf-test-view-role-${cluster_name}-${i}"
            local view_rb_name="perf-test-view-rb-${cluster_name}-${i}"

            cat <<EOF | kubectl apply -f - 2>&1 || log_warn "Failed to create view Role/RoleBinding for ${cluster_name}-${i}, continuing..."
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ${view_role_name}
  namespace: ${cluster_name}
  labels:
    test: "performance"
    cluster: "${cluster_name}"
    type: "view"
rules:
- apiGroups: ["view.open-cluster-management.io"]
  resources: ["managedclusterviews"]
  verbs: ["create", "get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ${view_rb_name}
  namespace: ${cluster_name}
  labels:
    test: "performance"
    cluster: "${cluster_name}"
    type: "view"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: ${view_role_name}
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: test-user
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: test-group
EOF
        done
    done

    local total_admin_clusterroles=$((num_clusters * num_admin_bindings_per_cluster))
    local total_view_clusterroles=$((num_clusters * num_view_bindings_per_cluster))
    local total_clusterroles=$((total_admin_clusterroles + total_view_clusterroles))
    local total_admin_clusterrolebindings=$((num_clusters * num_admin_bindings_per_cluster))
    local total_view_clusterrolebindings=$((num_clusters * num_view_bindings_per_cluster))
    local total_admin_roles=$((num_clusters * num_admin_bindings_per_cluster))
    local total_admin_rolebindings=$((num_clusters * num_admin_bindings_per_cluster))
    local total_view_roles=$((num_clusters * num_view_bindings_per_cluster))
    local total_view_rolebindings=$((num_clusters * num_view_bindings_per_cluster))

    log_info "  Created ${total_clusterroles} ClusterRoles (${total_admin_clusterroles} admin, ${total_view_clusterroles} view)"
    log_info "  Created ${total_admin_clusterrolebindings} admin ClusterRoleBindings and ${total_view_clusterrolebindings} view ClusterRoleBindings"
    log_info "  Created ${total_admin_roles} admin Roles and ${total_admin_rolebindings} admin RoleBindings"
    log_info "  Created ${total_view_roles} view Roles and ${total_view_rolebindings} view RoleBindings"
}

# Setup test on existing clusters
setup_on_existing_clusters() {
    log_info "Finding existing managed cluster namespaces..."

    # Get list of managed cluster namespaces
    local clusters=$(kubectl get namespaces -l open-cluster-management.io/cluster-name \
        -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

    if [ -z "$clusters" ]; then
        log_warn "No managed cluster namespaces found"
        log_warn "Looking for any namespaces that look like clusters..."
        clusters=$(kubectl get namespaces -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | grep -E '^(perf-test-cluster|cluster|local-cluster)' | head -${NUM_CLUSTERS} | tr '\n' ' ' || echo "")
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

    # Create discoverable ClusterRoles
    create_discoverable_clusterroles "$NUM_CLUSTERROLES"

    # Create admin/view RBAC bindings (tests processor_admin_view.go)
    create_admin_view_rbac "$NUM_ADMIN_BINDINGS_PER_CLUSTER" "$NUM_VIEW_BINDINGS_PER_CLUSTER" \
        "${cluster_array[@]:0:$NUM_CLUSTERS}"

    # Create test RBAC
    log_info "Creating test RBAC..."
    cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: userpermissions-viewer
rules:
- apiGroups: ["clusterview.open-cluster-management.io"]
  resources: ["userpermissions"]
  verbs: ["list", "get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: test-user-userpermissions
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: userpermissions-viewer
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
        create_test_clusterpermissions "$cluster" "$NAMESPACES_PER_CLUSTER" "$NUM_CLUSTERPERMISSIONS"
    done

    local total_admin_clusterroles=$((NUM_CLUSTERS * NUM_ADMIN_BINDINGS_PER_CLUSTER))
    local total_view_clusterroles=$((NUM_CLUSTERS * NUM_VIEW_BINDINGS_PER_CLUSTER))
    local total_clusterroles=$((total_admin_clusterroles + total_view_clusterroles))
    local total_clusterrolebindings=$((NUM_CLUSTERS * (NUM_ADMIN_BINDINGS_PER_CLUSTER + NUM_VIEW_BINDINGS_PER_CLUSTER)))
    local total_rolebindings_per_cluster=$((NUM_ADMIN_BINDINGS_PER_CLUSTER + NUM_VIEW_BINDINGS_PER_CLUSTER))

    log_info "Setup complete!"
    log_info "Total discoverable ClusterRoles: ${NUM_CLUSTERROLES}"
    log_info "Total ClusterPermissions created: $((NUM_CLUSTERS * NUM_CLUSTERPERMISSIONS))"
    log_info "Total namespace bindings: $((NUM_CLUSTERS * NAMESPACES_PER_CLUSTER))"
    log_info "Total admin/view ClusterRoles: ${total_clusterroles} (${total_admin_clusterroles} admin, ${total_view_clusterroles} view)"
    log_info "Total admin/view ClusterRoleBindings: ${total_clusterrolebindings}"
    log_info "Total admin/view Roles: $((NUM_CLUSTERS * total_rolebindings_per_cluster)) (1:1 with RoleBindings)"
    log_info "Total admin/view RoleBindings: $((NUM_CLUSTERS * total_rolebindings_per_cluster))"
}

# Measure performance for LIST operation
measure_list_performance() {
    log_info "Measuring LIST API performance with ${NUM_ITERATIONS} iterations..."

    local total_time=0
    local min_time=999999
    local max_time=0
    local clusterrole_count=0

    echo ""
    echo "Iteration | Response Time (ms) | ClusterRoles Returned"
    echo "----------|-------------------|----------------------"

    for ((i=1; i<=NUM_ITERATIONS; i++)); do
        local start_time=$(date +%s%N)

        local output=$(kubectl get userpermissions.clusterview.open-cluster-management.io \
            --as=test-user --as-group=test-group -o json 2>/dev/null || echo '{"items":[]}')

        local end_time=$(date +%s%N)
        local duration=$(( (end_time - start_time) / 1000000 ))

        if [ -n "$output" ]; then
            clusterrole_count=$(echo "$output" | jq '.items | length' 2>/dev/null || echo "0")
        fi

        printf "%9d | %17d | %21d\n" "$i" "$duration" "$clusterrole_count"

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
    log_info "LIST Performance Summary:"
    echo "  Average response time: ${avg_time} ms"
    echo "  Min response time: ${min_time} ms"
    echo "  Max response time: ${max_time} ms"
    echo "  ClusterRoles returned: ${clusterrole_count}"
    echo "  Throughput: $(echo "scale=2; 1000 / $avg_time" | bc 2>/dev/null || echo 'N/A') requests/second"
}

# Measure performance for GET operation (specific ClusterRole)
measure_get_performance() {
    log_info "Measuring GET API performance with ${NUM_ITERATIONS} iterations..."

    # Use the first ClusterRole for GET testing
    local clusterrole_name="perf-test-clusterrole-1"

    local total_time=0
    local min_time=999999
    local max_time=0
    local success_count=0

    echo ""
    echo "Iteration | Response Time (ms) | Success"
    echo "----------|-------------------|--------"

    for ((i=1; i<=NUM_ITERATIONS; i++)); do
        local start_time=$(date +%s%N)

        local output=$(kubectl get userpermissions.clusterview.open-cluster-management.io ${clusterrole_name} \
            --as=test-user --as-group=test-group -o json 2>/dev/null || echo '{}')

        local end_time=$(date +%s%N)
        local duration=$(( (end_time - start_time) / 1000000 ))

        local success="No"
        if [ -n "$output" ] && echo "$output" | jq -e '.metadata.name' >/dev/null 2>&1; then
            success="Yes"
            success_count=$((success_count + 1))
        fi

        printf "%9d | %17d | %7s\n" "$i" "$duration" "$success"

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
    log_info "GET Performance Summary:"
    echo "  Average response time: ${avg_time} ms"
    echo "  Min response time: ${min_time} ms"
    echo "  Max response time: ${max_time} ms"
    echo "  Successful requests: ${success_count}/${NUM_ITERATIONS}"
    echo "  Throughput: $(echo "scale=2; 1000 / $avg_time" | bc 2>/dev/null || echo 'N/A') requests/second"
}

# Measure performance
measure_performance() {
    measure_list_performance
    echo ""
    echo "=========================================="
    echo ""
    measure_get_performance
}

# Cleanup test resources
cleanup() {
    log_warn "Cleaning up test resources..."

    # Delete all ClusterPermissions matching the pattern
    local deleted_cps=0
    # Get all namespaces that have perf-test ClusterPermissions
    local namespaces=$(kubectl get clusterpermission --all-namespaces --no-headers 2>/dev/null | grep "perf-test-cp-" | awk '{print $1}' | sort -u)

    for ns in $namespaces; do
        for ((i=1; i<=20; i++)); do  # Support up to 20 ClusterPermissions per cluster
            if kubectl get clusterpermission "perf-test-cp-${i}" -n "$ns" --no-headers 2>/dev/null | grep -q "perf-test-cp-${i}"; then
                kubectl delete clusterpermission "perf-test-cp-${i}" -n "$ns" --ignore-not-found=true 2>/dev/null || true
                deleted_cps=$((deleted_cps + 1))
            fi
        done
    done

    log_info "Deleted ${deleted_cps} ClusterPermissions"

    # Delete admin/view ClusterRoleBindings
    local deleted_crbs=0
    local crb_output=$(kubectl delete clusterrolebinding -l test=performance --ignore-not-found=true 2>&1 || echo "")
    if [ -n "$crb_output" ]; then
        deleted_crbs=$(echo "$crb_output" | grep -c "deleted" || echo "0")
    fi
    log_info "Deleted ${deleted_crbs} ClusterRoleBindings"

    # Delete all ClusterRoles (both discoverable and admin/view)
    local deleted_crs=0
    local cr_output=$(kubectl delete clusterrole -l test=performance --ignore-not-found=true 2>&1 || echo "")
    if [ -n "$cr_output" ]; then
        deleted_crs=$(echo "$cr_output" | grep -c "deleted" || echo "0")
    fi
    log_info "Deleted ${deleted_crs} ClusterRoles"

    # Delete admin/view Roles and RoleBindings from cluster namespaces
    local deleted_roles=0
    local deleted_rbs=0

    # Get all managed cluster namespaces that might have our test resources
    local cluster_namespaces=$(kubectl get namespaces -l open-cluster-management.io/cluster-name \
        -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

    if [ -z "$cluster_namespaces" ]; then
        # Fallback: try to find namespaces that look like clusters
        cluster_namespaces=$(kubectl get namespaces -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | grep -E '^(perf-test-cluster|cluster|local-cluster)' | tr '\n' ' ' || echo "")
    fi

    for ns in $cluster_namespaces; do
        # Delete Roles with test=performance label
        local role_output=$(kubectl delete role -n "$ns" -l test=performance --ignore-not-found=true 2>&1 || echo "")
        if [ -n "$role_output" ]; then
            local count=$(echo "$role_output" | grep -c "deleted" || echo "0")
            deleted_roles=$((deleted_roles + count))
        fi

        # Delete RoleBindings with test=performance label
        local rb_output=$(kubectl delete rolebinding -n "$ns" -l test=performance --ignore-not-found=true 2>&1 || echo "")
        if [ -n "$rb_output" ]; then
            local count=$(echo "$rb_output" | grep -c "deleted" || echo "0")
            deleted_rbs=$((deleted_rbs + count))
        fi
    done

    log_info "Deleted ${deleted_roles} Roles and ${deleted_rbs} RoleBindings"

    # Delete test RBAC
    log_info "Deleting test RBAC..."
    kubectl delete clusterrolebinding test-user-userpermissions --ignore-not-found=true 2>/dev/null || true
    kubectl delete clusterrole userpermissions-viewer --ignore-not-found=true 2>/dev/null || true

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
Quick Performance Test for userpermissions API

Usage: $0 <command>

Commands:
  setup   - Create test ClusterRoles and ClusterPermissions in existing cluster namespaces
  test    - Run performance test (both LIST and GET operations)
  cleanup - Remove test ClusterRoles and ClusterPermissions
  full    - Run complete test (setup -> test -> cleanup)

Environment Variables:
  NUM_CLUSTERS            - Max number of clusters to use (default: 10)
  NAMESPACES_PER_CLUSTER  - Namespaces per cluster (default: 10)
  NUM_CLUSTERPERMISSIONS  - Number of ClusterPermission objects per cluster (default: 5)
  NUM_CLUSTERROLES        - Number of discoverable ClusterRoles to create (default: 5)
  NUM_ITERATIONS          - Test iterations (default: 10)
  KUBECONFIG              - Path to kubeconfig (default: ~/.kube/config)

Examples:
  # Quick test with defaults (10 clusters, 10 namespaces, 5 ClusterPermissions per cluster, 5 ClusterRoles)
  $0 full

  # Custom scale with more ClusterPermissions and ClusterRoles
  NUM_CLUSTERS=5 NAMESPACES_PER_CLUSTER=20 NUM_CLUSTERPERMISSIONS=10 NUM_CLUSTERROLES=10 NUM_ITERATIONS=50 $0 full

  # Stress test with many ClusterPermissions per cluster
  NUM_CLUSTERS=3 NAMESPACES_PER_CLUSTER=50 NUM_CLUSTERPERMISSIONS=20 NUM_CLUSTERROLES=15 $0 full

  # Use specific kubeconfig
  KUBECONFIG=/path/to/kubeconfig $0 full

Note:
  This script tests both LIST and GET operations for the userpermissions API.
  The LIST operation returns all ClusterRoles the user has bindings to.
  The GET operation retrieves a specific ClusterRole by name.
EOF
        ;;
esac
