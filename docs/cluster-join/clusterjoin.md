# Cluster Join Process

Mcm will constrain that one and only one cluster can be registered into a namespace (namespace for cluster). Two clusters cannot be put in the same namespace. Each cluster name must be unique globally.

When user wants to add a new managed cluster, klusterlet-connectionmanager will use bootstrap secret to create `clusterjoinrequest`, then api and controller will verify the request and return certificate if the request is approved. then klusterlet uses the certificate to talk to kube-apiserver.

## Cluster Join Request Spec

```yaml
apiVersion: mcm.ibm.com/v1alpha1
kind: ClusterJoinRequest
metadata:
  name: clusterjoin-8DXDn5dY
  resourceVersion: "43644"
spec:
  clusterName: c1
  clusterNameSpace: cn1
  csr:
    request: <CSR Data>
    usages: <Allowed Usages>
status:
  phase: Approved
  csrStatus:
    certificate: <Issued Certificate>
    conditions:
    - lastUpdateTime: "2019-12-10T02:33:12Z"
      message: This CSR was approved by cluster join controller.
      reason: ClusterJoinApprove
      type: Approved
```

In the `spec` section:

- `clusterName` specifies cluster name that you defined for managed cluster.
- `clusterNameSpace` specifies cluster namespace that the cluster should register.
- `csr` define the Certificate Signing Request data, it is `CertificateSigningRequestSpec` type.

In `status` section:

- `phase` specifies the cluster join request status. it should be `Approved` or `Denied`
- `csrStatus` define the Certificate Signing Request status. If request was approved, the controller will place the issued certificate in `csrStatus/certificate`.

## Work Flow

The whole flow of the process will be as follows:
![image](clusterjoin.png)
