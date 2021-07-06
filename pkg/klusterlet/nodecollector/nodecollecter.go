package nodecollector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	clusterapiv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prometheusconfig "github.com/prometheus/common/config"
	prometheusmodel "github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	updateJitterFactor                             = 0.25
	updateInterval                                 = 30
	resourceCore         clusterapiv1.ResourceName = "core"
	resourceSocket       clusterapiv1.ResourceName = "socket"
	resourceCoreWorker   clusterapiv1.ResourceName = "core_worker"
	resourceSocketWorker clusterapiv1.ResourceName = "socket_worker"
	resourceCPUWorker    clusterapiv1.ResourceName = "cpu_worker"
	defaultServer                                  = "https://prometheus-k8s.openshift-monitoring.svc:9091"
	defaultTokenFile                               = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	caConfigMapName                                = "ocm-controller-metrics-ca"

	// LabelNodeRolePrefix is a label prefix for node roles
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	LabelNodeRolePrefix = "node-role.kubernetes.io/"

	// NodeLabelRole specifies the role of a node
	NodeLabelRole = "kubernetes.io/role"
)

// LeaseUpdater is to update lease with certain period
type Collector interface {
	// Start starts a goroutine to update lease
	Start(ctx context.Context)
}

type queryResult struct {
	val      prometheusmodel.SampleValue
	isWorker bool
}

type resourceCollector struct {
	nodeLister         corev1lister.NodeLister
	nodeSynced         cache.InformerSynced
	kubeClient         kubernetes.Interface
	client             client.Client
	clusterName        string
	caData             []byte
	certPool           *x509.CertPool
	prometheusClient   prometheusv1.API
	tokenFile          string
	server             string
	componentNamespace string
}

func NewCollector(
	nodeInformer corev1informers.NodeInformer,
	kubeClient kubernetes.Interface,
	client client.Client, clusterName, componentNamespace string) Collector {
	return &resourceCollector{
		nodeLister:         nodeInformer.Lister(),
		nodeSynced:         nodeInformer.Informer().HasSynced,
		kubeClient:         kubeClient,
		client:             client,
		clusterName:        clusterName,
		server:             defaultServer,
		tokenFile:          defaultTokenFile,
		componentNamespace: componentNamespace,
	}
}

func (r *resourceCollector) Start(ctx context.Context) {
	if !cache.WaitForCacheSync(ctx.Done(), r.nodeSynced) {
		klog.Errorf("Failed to sync node cache")
		return
	}
	wait.JitterUntilWithContext(context.TODO(), r.reconcile, time.Duration(updateInterval)*time.Second, updateJitterFactor, true)
}

func (r *resourceCollector) reconcile(ctx context.Context) {
	clusterinfo := &clusterv1beta1.ManagedClusterInfo{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: r.clusterName, Name: r.clusterName}, clusterinfo)
	if err != nil {
		klog.Errorf("Failed to get cluster: %v", err)
	}

	if utils.ClusterIsOffLine(clusterinfo.Status.Conditions) {
		return
	}

	nodeList, err := r.getNodeList()
	if err != nil {
		klog.Errorf("Failed to get nodes: %v", err)
	}

	nodeList = r.updateCapacityByPrometheus(ctx, nodeList)

	// need sort the slices before compare using DeepEqual
	sort.SliceStable(nodeList, func(i, j int) bool { return nodeList[i].Name < nodeList[j].Name })
	if apiequality.Semantic.DeepEqual(nodeList, clusterinfo.Status.NodeList) {
		return
	}

	clusterinfo.Status.NodeList = nodeList
	err = r.client.Status().Update(ctx, clusterinfo)
	if err != nil {
		klog.Errorf("failed to update cluster resources: %v", err)
	}
}

func (r *resourceCollector) updateCapacityByPrometheus(ctx context.Context, nodes []clusterv1beta1.NodeStatus) []clusterv1beta1.NodeStatus {
	// get sockert/core from prometheus
	caData, err := r.getPrometheusCA(ctx)
	if err != nil {
		klog.Errorf("failed to get ca: %v", err)
		return nodes
	}
	if len(caData) == 0 {
		klog.Errorf("CA data does not exist")
		return nodes
	}

	if !reflect.DeepEqual(r.caData, caData) {
		apiClient, err := r.newPrometheusClient(caData)
		if err != nil {
			klog.Errorf("Failed to create prometheus client: %v", err)
			return nodes
		}

		r.caData = caData
		r.prometheusClient = apiClient
	}

	sockets, err := r.queryResource(ctx, r.prometheusClient, "machine_cpu_sockets")
	if err != nil {
		klog.Errorf("failed to query resource: %v", err)
	}

	for index := range nodes {
		if capacity, ok := sockets[nodes[index].Name]; ok {
			nodes[index].Capacity[resourceSocket] = *capacity
		}
	}

	return nodes
}

func (r *resourceCollector) getPrometheusCA(ctx context.Context) ([]byte, error) {
	cm, err := r.kubeClient.CoreV1().ConfigMaps(r.componentNamespace).Get(ctx, caConfigMapName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      caConfigMapName,
				Namespace: r.componentNamespace,
				Annotations: map[string]string{
					"service.alpha.openshift.io/inject-cabundle": "true",
				},
			},
			Data: map[string]string{
				"service-ca.crt": "",
			},
		}
		_, err := r.kubeClient.CoreV1().ConfigMaps(r.componentNamespace).Create(ctx, cm, metav1.CreateOptions{})
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	caString := cm.Data["service-ca.crt"]

	return []byte(caString), nil
}

func (r *resourceCollector) queryResource(ctx context.Context, client prometheusv1.API, name string) (map[string]*resource.Quantity, error) {
	results := map[string]*resource.Quantity{}
	result, warnings, err := client.Query(ctx, name, time.Now())
	if err != nil {
		return results, err
	}

	if len(warnings) != 0 {
		klog.Warningf("Get warning from prometheus service: %v", warnings)
	}

	if result.Type() != prometheusmodel.ValVector {
		return results, fmt.Errorf("the returrn data type is not correct: %v", result.Type())
	}

	vector := result.(prometheusmodel.Vector)
	if len(vector) == 0 {
		return results, nil
	}
	for _, v := range vector {
		nodeName := v.Metric["node"]
		// here needs unmarshal to new a quantity since NewQuantity is different from the value after unmarshal.
		data := strconv.FormatInt(int64(v.Value), 10)
		socket := &resource.Quantity{}
		_ = socket.UnmarshalJSON([]byte(data))
		results[string(nodeName)] = socket
	}

	return results, nil
}

func (r *resourceCollector) newPrometheusClient(caData []byte) (prometheusv1.API, error) {
	var err error
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// setup transport CA
	if r.certPool == nil {
		r.certPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
	}

	// AppendCertsFromPEM can update ca by subject of ca since ca is stored in a map with subject as key.
	if !r.certPool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("no cert found in ca file")
	}

	transport.TLSClientConfig = &tls.Config{RootCAs: r.certPool, MinVersion: tls.VersionTLS12}

	// read token from token files
	client, err := prometheusapi.NewClient(prometheusapi.Config{
		Address:      r.server,
		RoundTripper: prometheusconfig.NewBearerAuthFileRoundTripper(r.tokenFile, transport),
	})
	if err != nil {
		return nil, err
	}

	return prometheusv1.NewAPI(client), nil
}

func (r *resourceCollector) getNodeList() ([]clusterv1beta1.NodeStatus, error) {
	var nodeList []clusterv1beta1.NodeStatus
	nodes, err := r.nodeLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		nodeStatus := clusterv1beta1.NodeStatus{
			Name:       node.Name,
			Labels:     map[string]string{},
			Capacity:   clusterv1beta1.ResourceList{},
			Conditions: []clusterv1beta1.NodeCondition{},
		}

		// The roles are determined by looking for:
		// * a node-role.kubernetes.io/<role>="" label
		// * a kubernetes.io/role="<role>" label
		for k, v := range node.Labels {
			switch k {
			case NodeLabelRole, corev1.LabelFailureDomainBetaRegion, corev1.LabelFailureDomainBetaZone, corev1.LabelInstanceType, corev1.LabelInstanceTypeStable:
				nodeStatus.Labels[k] = v
			}
			if strings.HasPrefix(k, LabelNodeRolePrefix) {
				nodeStatus.Labels[k] = v
			}
		}

		// append capacity of cpu and memory
		for k, v := range node.Status.Capacity {
			switch {
			case k == corev1.ResourceCPU:
				nodeStatus.Capacity[clusterv1beta1.ResourceCPU] = v
			case k == corev1.ResourceMemory:
				nodeStatus.Capacity[clusterv1beta1.ResourceMemory] = v
			}
		}

		// append condition of NodeReady
		readyCondition := clusterv1beta1.NodeCondition{
			Type: corev1.NodeReady,
		}
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				readyCondition.Status = condition.Status
				break
			}
		}
		nodeStatus.Conditions = append(nodeStatus.Conditions, readyCondition)

		nodeList = append(nodeList, nodeStatus)
	}
	return nodeList, nil
}
