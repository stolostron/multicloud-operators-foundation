package resourcecollector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"time"

	clusterclient "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterapiv1 "github.com/open-cluster-management/api/cluster/v1"
	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prometheusconfig "github.com/prometheus/common/config"
	prometheusmodel "github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	updateJitterFactor                             = 0.25
	updateInterval                                 = 30
	resourceCore         clusterapiv1.ResourceName = "core"
	resourceSocket       clusterapiv1.ResourceName = "socket"
	resourceCoreWorker   clusterapiv1.ResourceName = "core_worker"
	resourceSocketWorker clusterapiv1.ResourceName = "socket_worker"
	defaultServer                                  = "https://prometheus-k8s.openshift-monitoring.svc:9091"
	defaultTokenFile                               = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	caConfigMapName                                = "ocm-controller-metrics-ca"
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
	kubeClient         kubernetes.Interface
	clusterClient      clusterclient.Interface
	clusterName        string
	caData             []byte
	tokenFile          string
	server             string
	componentNamespace string
}

func NewCollector(kubeClient kubernetes.Interface, clusterClient clusterclient.Interface, clusterName, componentNamespace string) Collector {
	return &resourceCollector{
		kubeClient:         kubeClient,
		clusterClient:      clusterClient,
		clusterName:        clusterName,
		server:             defaultServer,
		tokenFile:          defaultTokenFile,
		componentNamespace: componentNamespace,
	}
}

func (r *resourceCollector) Start(ctx context.Context) {
	wait.JitterUntilWithContext(context.TODO(), r.reconcile, time.Duration(updateInterval)*time.Second, updateJitterFactor, true)
}

func (r *resourceCollector) reconcile(ctx context.Context) {
	caData, err := r.getPrometheusCA(ctx)
	if err != nil {
		klog.Errorf("failed to get ca: %v", err)
	}
	if len(caData) == 0 {
		klog.Errorf("CA data does not exist")
		return
	}

	apiClient, err := r.newPrometheusClient(caData)
	if err != nil {
		klog.Errorf("Failed to create prometheus client: %v", err)
		return
	}

	cluster, err := r.clusterClient.ClusterV1().ManagedClusters().Get(ctx, r.clusterName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get cluster: %v", err)
	}
	capacity := cluster.DeepCopy().Status.Capacity
	totalCore, workerCore, err := r.queryResource(ctx, apiClient, "machine_cpu_cores")
	switch {
	case err != nil:
		klog.Errorf("failed to query resource: %v", err)
	case totalCore != nil:
		capacity[resourceCore] = *totalCore
		capacity[resourceCoreWorker] = *workerCore
	}

	totalSocket, workerSocket, err := r.queryResource(ctx, apiClient, "machine_cpu_sockets")
	switch {
	case err != nil:
		klog.Errorf("failed to query resource: %v", err)
	case totalSocket != nil:
		capacity[resourceSocket] = *totalSocket
		capacity[resourceSocketWorker] = *workerSocket
	}

	if apiequality.Semantic.DeepEqual(capacity, cluster.Status.Capacity) {
		return
	}

	cluster.Status.Capacity = capacity
	_, err = r.clusterClient.ClusterV1().ManagedClusters().UpdateStatus(ctx, cluster, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("failed to update cluster resources")
	}
}

func (r *resourceCollector) getPrometheusCA(ctx context.Context) ([]byte, error) {
	if len(r.caData) > 0 {
		return r.caData, nil
	}

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
	if len(caString) > 0 {
		r.caData = []byte(caString)
	}

	return r.caData, nil
}

func (r *resourceCollector) queryResource(ctx context.Context, client prometheusv1.API, name string) (*resource.Quantity, *resource.Quantity, error) {
	results := []queryResult{}
	result, warnings, err := client.Query(ctx, name, time.Now())
	if err != nil {
		return nil, nil, err
	}

	if len(warnings) != 0 {
		klog.Warningf("Get warning from prometheus service: %v", warnings)
	}

	if result.Type() != prometheusmodel.ValVector {
		return nil, nil, fmt.Errorf("the returrn data type is not correct: %v", result.Type())
	}

	vector := result.(prometheusmodel.Vector)
	if len(vector) == 0 {
		return nil, nil, nil
	}
	for _, v := range vector {
		res := queryResult{val: v.Value}
		isWorker, err := r.isWorker(ctx, v.Metric)
		if err != nil {
			klog.Errorf("failed to get node: %v", err)
			continue
		}
		res.isWorker = isWorker
		results = append(results, res)
	}

	var total, worker prometheusmodel.SampleValue
	for _, res := range results {
		total = total + res.val
		if res.isWorker {
			worker = worker + res.val
		}
	}

	if total == 0 {
		return nil, nil, nil
	}

	return resource.NewQuantity(int64(total), resource.DecimalSI), resource.NewQuantity(int64(worker), resource.DecimalSI), nil
}

func (r *resourceCollector) isWorker(ctx context.Context, metric prometheusmodel.Metric) (bool, error) {
	nodeName := metric["node"]
	node, err := r.kubeClient.CoreV1().Nodes().Get(ctx, string(nodeName), metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if node.Labels == nil {
		return false, nil
	}

	if _, ok := node.Labels["node-role.kubernetes.io/worker"]; ok {
		return true, nil
	}

	return false, nil
}

func (r *resourceCollector) newPrometheusClient(caData []byte) (prometheusv1.API, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// setup transport CA
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("no cert found in ca file")
	}

	transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}

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
