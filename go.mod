module github.com/stolostron/multicloud-operators-foundation

go 1.14

// Determined by go.mod in github.com/openshift/hive
replace (
	github.com/Azure/go-autorest => github.com/tombuildsstuff/go-autorest v14.0.1-0.20200416184303-d4e299a3c04a+incompatible
	github.com/Azure/go-autorest/autorest => github.com/tombuildsstuff/go-autorest/autorest v0.10.1-0.20200416184303-d4e299a3c04a
	github.com/Azure/go-autorest/autorest/azure/auth => github.com/tombuildsstuff/go-autorest/autorest/azure/auth v0.4.3-0.20200416184303-d4e299a3c04a
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20190821174549-a2a477909c1d
	github.com/terraform-providers/terraform-provider-aws => github.com/openshift/terraform-provider-aws v1.60.1-0.20200630224953-76d1fb4e5699
	github.com/terraform-providers/terraform-provider-azurerm => github.com/openshift/terraform-provider-azurerm v1.40.1-0.20200707062554-97ea089cc12a
	github.com/terraform-providers/terraform-provider-ignition/v2 => github.com/community-terraform-providers/terraform-provider-ignition/v2 v2.1.0
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200506073438-9d49428ff837
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20200120114645-8a9592f1f87b
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20200526112135-319a35b2e38e
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1 // ensure compatible between controller-runtime and kube-openapi
	github.com/open-cluster-management/api => open-cluster-management.io/api v0.2.1-0.20201126023000-353dd8370f4d
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20200918101923-1e4c94603efe
	k8s.io/client-go => k8s.io/client-go v0.19.0
)

require (
	github.com/emicklei/go-restful v2.13.0+incompatible
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/go-openapi/spec v0.19.8
	github.com/gogo/protobuf v1.3.2
	github.com/google/go-cmp v0.5.1 // indirect
	github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/metal3-io/baremetal-operator v0.0.0
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/open-cluster-management/api v0.0.0-00010101000000-000000000000
	github.com/openshift/build-machinery-go v0.0.0-20200819073603-48aa266c95f7
	github.com/openshift/client-go v0.0.0-20200827190008-3062137373b5
	github.com/openshift/custom-resource-status v0.0.0-20200602122900-c002fd1547ca
	github.com/openshift/hive v1.0.17
	github.com/prometheus/common v0.10.0
	github.com/smartystreets/goconvey v1.6.4
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/net v0.0.0-20201021035429-f5854403a974
	k8s.io/api v0.19.0
	k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/apiserver v0.19.0
	k8s.io/cli-runtime v0.19.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.19.0
	k8s.io/component-base v0.19.0
	k8s.io/klog v1.0.0
	k8s.io/kube-aggregator v0.19.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	sigs.k8s.io/controller-runtime v0.6.2
)
