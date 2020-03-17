module github.com/open-cluster-management/multicloud-operators-foundation

go 1.12

replace (
	// Pin kube version to 1.13.1
	k8s.io/api => k8s.io/api v0.0.0-20181213150558-05914d821849
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20181213153335-0fe22c71c476
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20181213151703-3ccfe8365421
	k8s.io/client-go => k8s.io/client-go v0.0.0-20181213151034-8d9ed539ba31
	k8s.io/klog => k8s.io/klog v0.2.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20181109181836-c59034cc13d5
	sigs.k8s.io/yaml => sigs.k8s.io/yaml v1.1.0
)

require (
	bitbucket.org/ww/goautoneg v0.0.0-20120707110453-75cd24fc2f2c // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/NYTimes/gziphandler v0.0.0-20180221000450-2600fb119af9 // indirect
	github.com/PuerkitoBio/purell v0.0.0-20161115024942-0bcb03f4b4d0 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973 // indirect
	github.com/buger/jsonparser v0.0.0-20180808090653-f4dd9f5a6b44 // indirect
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/emicklei/go-restful v2.11.1+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.4
	github.com/gophercloud/gophercloud v0.8.0 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v0.0.0-20180604122856-c225b8c3b01f // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.11.2 // indirect
	github.com/hashicorp/golang-lru v0.0.0-20180201235237-0fb14efe8c47 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.0.0-20180608140156-9316a62528ac // indirect
	github.com/inconshreveable/mousetrap v0.0.0-20141017200713-76626ae9c91c // indirect
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/json-iterator/go v0.0.0-20180701071628-ab8a2e0c74be // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/mailru/easyjson v0.0.0-20180723221831-d5012789d665 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/metal3-io/baremetal-operator v0.0.0-20200228072510-dfd5fea95b0a
	github.com/mongodb/mongo-go-driver v0.0.0-20180808171322-9b26cbb1a61a
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/openshift/client-go v0.0.0-20190923180330-3b6373338c9b
	github.com/openshift/custom-resource-status v0.0.0-20190822192428-e62f2f3b79f3
	github.com/openshift/hive v0.0.0-20200302220132-867d7fb189bf
	github.com/operator-framework/operator-sdk v0.15.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.17.2
	k8s.io/apiextensions-apiserver v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/apiserver v0.0.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/cluster-registry v0.0.0-20180711215825-1e7f92e96c20
	k8s.io/component-base v0.0.0
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/kubernetes v1.17.3
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	sigs.k8s.io/controller-runtime v0.4.0
)
