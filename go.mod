module github.com/open-cluster-management/multicloud-operators-foundation

go 1.16

replace k8s.io/client-go => k8s.io/client-go v0.20.0

// Determined by go.mod in github.com/openshift/hive
// from installer
replace (
	github.com/Azure/go-autorest/autorest/adal => github.com/Azure/go-autorest/autorest/adal v0.8.2
	github.com/Azure/go-autorest/autorest/azure/cli => github.com/Azure/go-autorest/autorest/azure/cli v0.3.1
	github.com/Azure/go-autorest/autorest/date => github.com/Azure/go-autorest/autorest/date v0.2.0
	github.com/Azure/go-autorest/autorest/validation => github.com/Azure/go-autorest/autorest/validation v0.2.1-0.20191028180845-3492b2aff503
	github.com/apparentlymart/go-cidr => github.com/apparentlymart/go-cidr v1.0.1
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.0
	github.com/hashicorp/go-plugin => github.com/hashicorp/go-plugin v1.2.2
	github.com/kubevirt/terraform-provider-kubevirt => github.com/nirarg/terraform-provider-kubevirt v0.0.0-20201222125919-101cee051ed3
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20190821174549-a2a477909c1d
	github.com/openshift/hive/apis => github.com/openshift/hive/apis v0.0.0-20210415080537-ea6f0a2dd76c
	github.com/terraform-providers/terraform-provider-aws => github.com/openshift/terraform-provider-aws v1.60.1-0.20200630224953-76d1fb4e5699
	github.com/terraform-providers/terraform-provider-azurerm => github.com/openshift/terraform-provider-azurerm v1.40.1-0.20200707062554-97ea089cc12a
	github.com/terraform-providers/terraform-provider-ignition/v2 => github.com/community-terraform-providers/terraform-provider-ignition/v2 v2.1.0
	github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.7
	google.golang.org/api => google.golang.org/api v0.25.0
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
	kubevirt.io/client-go => kubevirt.io/client-go v0.29.0
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20201022175424-d30c7a274820
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20201016155852-4090a6970205
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20201116051540-155384b859c5
)

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1 // ensure compatible between controller-runtime and kube-openapi

replace github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20200918101923-1e4c94603efe

require (
	github.com/emicklei/go-restful v2.14.2+incompatible
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/spec v0.19.9
	github.com/gogo/protobuf v1.3.2
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/metal3-io/baremetal-operator v0.0.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/open-cluster-management/addon-framework v0.0.0-20210506065618-51dcadda8be6
	github.com/open-cluster-management/api v0.0.0-20210409125704-06f2aec1a73f
	github.com/openshift/build-machinery-go v0.0.0-20210209125900-0da259a2c359
	github.com/openshift/client-go v0.0.0-20201020074620-f8fd44879f7c
	github.com/openshift/custom-resource-status v0.0.0-20200602122900-c002fd1547ca
	github.com/openshift/hive v1.1.1
	github.com/openshift/hive/apis v0.0.0
	github.com/openshift/library-go v0.0.0-20210330121802-ebbc677c82a5
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.10.0
	github.com/smartystreets/goconvey v1.6.4
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/apiserver v0.20.0
	k8s.io/cli-runtime v0.20.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.20.0
	k8s.io/component-base v0.20.5
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.8.0
	k8s.io/kube-aggregator v0.20.0
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	sigs.k8s.io/controller-runtime v0.7.0
)
