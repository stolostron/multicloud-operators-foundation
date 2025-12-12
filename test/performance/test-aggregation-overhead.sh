#!/bin/bash

# UserPermission API Aggregation Overhead Test
#
# This script measures the performance overhead introduced by Kubernetes APIServer aggregation
# by comparing direct proxyserver access vs access through the K8s aggregation layer.
#
# Prerequisites:
# - kubectl configured and connected to your cluster
# - ocm-proxyserver running in multicluster-engine namespace
# - userpermissions API available
# - Valid user token with permissions to access userpermissions API
#
# Usage:
#   TOKEN="your-token-here" bash test-aggregation-overhead.sh
#
# Configuration:
#   TOKEN - Bearer token for authentication (required)
#   NUM_ITERATIONS - Number of test iterations (default: 10)

set -e

# Configuration
NUM_ITERATIONS=${NUM_ITERATIONS:-10}
TOKEN=${TOKEN:-""}

# Validate token
if [ -z "$TOKEN" ]; then
    echo "[ERROR] TOKEN environment variable is required"
    echo "Usage: TOKEN=\"your-token\" bash test-aggregation-overhead.sh"
    exit 1
fi

echo "======================================"
echo "APIServer Aggregation Overhead Test"
echo "======================================"
echo ""
echo "Configuration:"
echo "  Iterations: $NUM_ITERATIONS"
echo "  Token: ${TOKEN:0:20}..."
echo ""

# Test 1: Direct ProxyServer Access
echo "[Test 1] Direct ProxyServer Access (bypassing aggregation)"
echo "==========================================================="
kubectl run perf-test-aggregation --image=curlimages/curl:latest --rm -i --restart=Never -- sh -c "
  echo ''
  TOTAL=0
  for i in 1 2 3 4 5 6 7 8 9 10; do
    RESULT=\$(curl -k -s -o /dev/null -w '%{time_total},%{http_code}' -H 'Authorization: Bearer $TOKEN' https://ocm-proxyserver.multicluster-engine.svc.cluster.local:443/apis/clusterview.open-cluster-management.io/v1alpha1/userpermissions)
    TIME_SEC=\$(echo \$RESULT | cut -d',' -f1)
    HTTP=\$(echo \$RESULT | cut -d',' -f2)
    # Convert seconds to milliseconds
    TIME_MS=\$(echo \"\$TIME_SEC * 1000\" | sed 's/\\.//g' | sed 's/^0*//' | cut -c1-2)
    echo \"Iteration \$i: \${TIME_SEC}s = ~\${TIME_MS}ms (HTTP \$HTTP)\"
    TOTAL=\$((TOTAL + TIME_MS))
  done
  AVG=\$((TOTAL / 10))
  echo \"\"
  echo \"Direct ProxyServer Average: \${AVG}ms\"
  "

echo ""
echo "[Test 2] K8s APIServer Aggregation"
echo "===================================="

# Unset proxy settings that might interfere
unset HTTP_PROXY HTTPS_PROXY

TOTAL=0
for i in $(seq 1 $NUM_ITERATIONS); do
    START=$(date +%s%N)
    kubectl --token="$TOKEN" get userpermissions.clusterview.open-cluster-management.io -o json >/dev/null 2>&1
    END=$(date +%s%N)
    DURATION=$(( ($END - $START) / 1000000 ))
    echo "Iteration $i: ${DURATION}ms"
    TOTAL=$(( $TOTAL + $DURATION ))
done

AVG=$(( $TOTAL / NUM_ITERATIONS ))
echo ""
echo "K8s APIServer Aggregation Average: ${AVG}ms"
echo ""

echo "======================================"
echo "Results Summary"
echo "======================================"
echo "This test compares the performance of:"
echo "  1. Direct access to ocm-proxyserver (bypassing K8s aggregation)"
echo "  2. Access through K8s APIServer aggregation layer"
echo ""
echo "A significant performance difference indicates that the bottleneck"
echo "is in the K8s aggregation layer, not in the proxyserver code itself."
echo ""
echo "See the output above for detailed timing results."
echo "======================================"
