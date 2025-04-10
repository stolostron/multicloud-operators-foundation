module github.com/stolostron/multicloud-operators-foundation

go 1.20

replace github.com/openshift/hive/apis => github.com/openshift/hive/apis v0.0.0-20231019152019-e3f9aed5bf2a

// CVE-2025-22868
replace golang.org/x/oauth2 => github.com/openshift/golang-oauth2 v0.26.1-0.20250310184649-06a918c6239d

// CVE-2025-22869
replace golang/x/crypto => github.com/openshift/golang-crypto v0.33.1-0.20250310193910-9003f682e581

// from installer
replace (
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20230531194024-8dde0991ffdd
	github.com/metal3-io/baremetal-operator/apis => github.com/openshift/baremetal-operator/apis v0.0.0-20230531194024-8dde0991ffdd
	github.com/metal3-io/baremetal-operator/pkg/hardwareutils => github.com/openshift/baremetal-operator/pkg/hardwareutils v0.0.0-20230531194024-8dde0991ffdd
	github.com/terraform-providers/terraform-provider-nutanix => github.com/nutanix/terraform-provider-nutanix v1.5.0
	k8s.io/cloud-provider-vsphere => github.com/openshift/cloud-provider-vsphere v1.19.1-0.20211222185833-7829863d0558
	sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v0.4.5
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200929152424-eab2e087f366 // Indirect dependency through MAO from cluster API providers
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20210626224711-5d94c794092f // Indirect dependency through MAO from cluster API providers
)

// from installer as part of https://github.com/openshift/installer/pull/4350
// Prevent the following modules from upgrading version as result of terraform-provider-kubernetes module
// The following modules need to be locked to compile correctly with terraform-provider-azure and terraform-provider-google
replace (
	github.com/apparentlymart/go-cidr => github.com/apparentlymart/go-cidr v1.0.1
	github.com/hashicorp/go-plugin => github.com/hashicorp/go-plugin v1.2.2
	github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.7
	google.golang.org/api => google.golang.org/api v0.25.0
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
)

// needed because otherwise v12.0.0 is picked up as a more recent version
replace k8s.io/client-go => k8s.io/client-go v0.27.2

// needed for fixing CVE-2020-29529
replace github.com/hashicorp/go-slug => github.com/hashicorp/go-slug v0.5.0

// needed for fixing CVE-2020-26160
// taken from https://github.com/openshift/installer/blob/21cd5218bb58288cd7b03018b9a2513aca3a13a5/terraform/providers/ibm/go.mod
replace github.com/dgrijalva/jwt-go v3.2.0+incompatible => github.com/golang-jwt/jwt v3.2.1+incompatible

// needed for fixing CVE-2018-1099
exclude (
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	go.etcd.io/etcd v0.0.0-20200513171258-e048e166ab9c
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200819165624-17cef6e3e9d5
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200910180754-dd1b699fc489
)

replace cloud.google.com/go => cloud.google.com/go v0.100.2

require (
	github.com/emicklei/go-restful v2.16.0+incompatible
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v1.2.4
	github.com/gogo/protobuf v1.3.2
	github.com/mattbaird/jsonpatch v0.0.0-20230413205102-771768614e91
	github.com/onsi/ginkgo/v2 v2.9.5
	github.com/onsi/gomega v1.27.7
	github.com/openshift/api v0.0.0-20230720094506-afcbe27aec7c
	github.com/openshift/build-machinery-go v0.0.0-20230306181456-d321ffa04533
	github.com/openshift/client-go v0.0.0-20230503144108-75015d2347cb
	github.com/openshift/hive v1.1.17-0.20231019152019-e3f9aed5bf2a
	github.com/openshift/hive/apis v0.0.0
	github.com/openshift/library-go v0.0.0-20230508110756-9b7abe2c9cbf
	github.com/prometheus/client_golang v1.15.1
	github.com/prometheus/common v0.42.0
	github.com/smartystreets/goconvey v1.8.1
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	github.com/stolostron/cluster-lifecycle-api v0.0.0-20230829070855-cd9b187cca82
	github.com/stretchr/testify v1.8.4
	golang.org/x/net v0.33.0
	k8s.io/api v0.27.2
	k8s.io/apiextensions-apiserver v0.27.2
	k8s.io/apimachinery v0.27.2
	k8s.io/apiserver v0.27.2
	k8s.io/cli-runtime v0.27.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.27.2
	k8s.io/component-base v0.27.2
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.100.1
	k8s.io/kube-aggregator v0.27.2
	k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f
	open-cluster-management.io/addon-framework v0.8.0
	open-cluster-management.io/api v0.12.0
	sigs.k8s.io/controller-runtime v0.15.0
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.0 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.4.0 // indirect
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.10.1 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/cel-go v0.12.6 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/huandu/xstrings v1.3.3 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/openshift/custom-resource-status v1.1.3-0.20220503160415-f2fdb4999d87 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/smarty/assertions v1.15.0 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/cobra v1.6.1 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.7 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.7 // indirect
	go.etcd.io/etcd/client/v3 v3.5.7 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.44.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.44.0 // indirect
	go.opentelemetry.io/otel v1.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.19.0 // indirect
	go.opentelemetry.io/otel/sdk v1.19.0 // indirect
	go.opentelemetry.io/otel/trace v1.19.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.starlark.net v0.0.0-20200306205701-8dd3e2ee1dd5 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/oauth2 v0.10.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/term v0.27.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	gomodules.xyz/jsonpatch/v2 v2.3.0 // indirect
	google.golang.org/genproto v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/grpc v1.58.3 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	helm.sh/helm/v3 v3.11.1 // indirect
	k8s.io/gengo v0.0.0-20220902162205-c0856e24416d // indirect
	k8s.io/kms v0.27.2 // indirect
	k8s.io/utils v0.0.0-20230505201702-9f6742963106 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.1.2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.13.2 // indirect
	sigs.k8s.io/kustomize/kyaml v0.14.1 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)
