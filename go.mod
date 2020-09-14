module github.com/open-cluster-management/multicloud-operators-foundation

go 1.14

// Determined by go.mod in github.com/openshift/hive
replace (
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0 // Pin non-versioned import to v22.0.0
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200206190020-71b826cc0f0a // Use OpenShift fork
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20190821174549-a2a477909c1d // Pin OpenShift fork
	github.com/terraform-providers/terraform-provider-aws => github.com/openshift/terraform-provider-aws v1.60.1-0.20200526184553-1a716dcc0fa8 // Pin to openshift fork with tag v2.60.0-openshift-1
	github.com/terraform-providers/terraform-provider-azurerm => github.com/openshift/terraform-provider-azurerm v1.41.1-openshift-3 // Pin to openshift fork with IPv6 fixes
	go.etcd.io/etcd => go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738 // Pin to version used by k8s.io/apiserver
	google.golang.org/grpc => google.golang.org/grpc v1.23.1 // Pin to version used by k8s.io/apiservers
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200506073438-9d49428ff837 // Pin OpenShift fork
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20200120114645-8a9592f1f87b // Pin OpenShift fork
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20200526112135-319a35b2e38e // Pin OpenShift fork
)

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1 // ensure compatible between controller-runtime and kube-openapi

require (
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/coreos/go-systemd v0.0.0 // indirect
	github.com/emicklei/go-restful v2.13.0+incompatible
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.2.1-0.20200730175230-ee2de8da5be6
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/go-openapi/spec v0.19.8
	github.com/gogo/protobuf v1.3.1
	github.com/google/go-cmp v0.5.1 // indirect
	github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20191106031601-ce3c9ade29de // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.1 // indirect
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/metal3-io/baremetal-operator v0.0.0
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/open-cluster-management/api v0.0.0-20200903203421-64b667f5455c
	github.com/openshift/api v0.0.0-20200720083901-0c4b3ae5f5df // indirect
	github.com/openshift/build-machinery-go v0.0.0-20200819073603-48aa266c95f7
	github.com/openshift/client-go v0.0.0-20200623090625-83993cebb5ae
	github.com/openshift/custom-resource-status v0.0.0-20200602122900-c002fd1547ca
	github.com/openshift/hive v1.0.0
	github.com/prometheus/common v0.10.0
	github.com/smartystreets/goconvey v1.6.4
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	go.etcd.io/etcd v3.3.22+incompatible // indirect
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	k8s.io/api v0.19.0
	k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/apiserver v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/component-base v0.19.0
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	sigs.k8s.io/controller-runtime v0.6.2
)
