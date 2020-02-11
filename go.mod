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
	github.com/coreos/etcd v0.0.0-20180724164832-fca8add78a9d // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20180511133405-39ca1b05acc7 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/docker v0.0.0-20180504203344-eeea1e37a116 // indirect
	github.com/elazarl/go-bindata-assetfs v0.0.0-20170227212728-30f82fa23fd8 // indirect
	github.com/emicklei/go-restful v0.0.0-20180701195719-3eb9738c1697
	github.com/emicklei/go-restful-swagger12 v0.0.0-20170208215640-dcef7f557305 // indirect
	github.com/evanphx/json-patch v0.0.0-20180322033437-afac545df32f // indirect
	github.com/go-openapi/jsonpointer v0.0.0-20180322222829-3a0015ad55fa // indirect
	github.com/go-openapi/jsonreference v0.0.0-20180322222742-3fb327e6747d // indirect
	github.com/go-openapi/spec v0.0.0-20180710175419-bce47c9386f9
	github.com/go-openapi/swag v0.0.0-20180703152219-2b0bd4f193d0 // indirect
	github.com/go-stack/stack v0.0.0-20171112031402-259ab82a6cad // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v0.0.0-20180717141946-636bf0302bc9 // indirect
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/google/btree v0.0.0-20180124185431-e89373fe6b4a // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.0.0-20180519185700-7c663266750e // indirect
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
	github.com/matttproud/golang_protobuf_extensions v0.0.0-20160424113007-c12348ce28de // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/mongodb/mongo-go-driver v0.0.0-20180808171322-9b26cbb1a61a
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/openshift/api v0.0.0-20190401220125-3a6077f1f910 // indirect
	github.com/openshift/client-go v0.0.0-20190401163519-84c2b942258a
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c // indirect
	github.com/peterbourgon/diskv v0.0.0-20170814173558-5f041e8faa00 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/prometheus/client_golang v0.0.0-20160817154824-c5b7fccd2042 // indirect
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910 // indirect
	github.com/prometheus/common v0.0.0-20180518154759-7600349dcfe1 // indirect
	github.com/prometheus/procfs v0.0.0-20180725123919-05ee40e3a273 // indirect
	github.com/sirupsen/logrus v0.0.0-20190402161407-8bdbc7bcc01d // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/spf13/cobra v0.0.0-20180427134550-ef82de70bb3f
	github.com/spf13/pflag v0.0.0-20180412120913-583c0c0531f0
	github.com/tidwall/pretty v1.0.0 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/ugorji/go v1.1.7 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	go.etcd.io/bbolt v1.3.3 // indirect
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	golang.org/x/tools v0.0.0-20191209225234-22774f7dae43 // indirect
	gopkg.in/inf.v0 v0.0.0-20180326172332-d2d2541c53f1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3 // indirect
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/api v0.0.0-20181213150558-05914d821849
	k8s.io/apiextensions-apiserver v0.0.0-20181213153335-0fe22c71c476
	k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93
	k8s.io/apiserver v0.0.0-20181213151703-3ccfe8365421
	k8s.io/client-go v0.0.0-20181213151034-8d9ed539ba31
	k8s.io/cluster-registry v0.0.0-20180711215825-1e7f92e96c20
	k8s.io/helm v0.0.0-20180919182530-2e55dbe1fdb5
	k8s.io/klog v0.3.0
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d
	k8s.io/utils v0.0.0-20190923111123-69764acb6e8e // indirect
	sigs.k8s.io/yaml v0.0.0-20181102190223-fd68e9863619 // indirect
)
