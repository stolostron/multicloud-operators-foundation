# PreReqs: kubectl installed with a valid KUBECONFIG for the target cluster.

# Patch the pre-existing resource with a new image source/version.  
pod_name=$(kubectl -n mvp-demo get po | grep "$IMAGE_DEPLOYED_NAME" | awk '{print $1}')
kubectl -n mvp-demo patch po $pod_name --type='json' -p='[{"op": "replace", "path": "/spec/containers/0/image", "value":"'$1'"}]'
