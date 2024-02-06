package getter

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"net/http"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	"strconv"
)

type LogProxyGetter struct {
	AddonClient        addonv1alpha1client.Interface
	KubeClient         kubernetes.Interface
	ProxyServiceHost   string
	ProxyServiceCAFile string
}

func NewLogProxyGetter(addonClient addonv1alpha1client.Interface,
	KubeClient kubernetes.Interface,
	host, caFile string) *LogProxyGetter {
	return &LogProxyGetter{
		AddonClient:        addonClient,
		KubeClient:         KubeClient,
		ProxyServiceHost:   host,
		ProxyServiceCAFile: caFile,
	}
}

func (c *LogProxyGetter) ProxyServiceAvailable(ctx context.Context, clusterName string) (bool, error) {
	_, err := c.AddonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).Get(ctx, helpers.MsaAddonName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get msaCma. %v", err)
	}
	_, err = c.AddonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).Get(ctx, helpers.ClusterProxyAddonName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get clusterProxyCma. %v", err)
	}

	return true, nil
}

func (c *LogProxyGetter) NewHandler(ctx context.Context, clusterName, podNamespace, podName, containerName string) (*Handler, error) {
	logTokenSecret, err := c.KubeClient.CoreV1().Secrets(clusterName).Get(ctx, helpers.LogManagedServiceAccountName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("faield to get log token secret in cluster %s. %v", clusterName, err)
	}
	clusterProxyCfg := &rest.Config{
		Host: fmt.Sprintf("https://%s/%s", c.ProxyServiceHost, clusterName),
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: c.ProxyServiceCAFile,
		},
		BearerToken: string(logTokenSecret.Data["token"]),
	}
	clusterProxyKubeClient, err := kubernetes.NewForConfig(clusterProxyCfg)
	if err != nil {
		return nil, err
	}
	return &Handler{
		logClient:     clusterProxyKubeClient,
		podName:       podName,
		podNamespace:  podNamespace,
		containerName: containerName,
	}, nil
}

type Handler struct {
	logClient                            kubernetes.Interface
	podNamespace, podName, containerName string
}

func (c *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	writeResponseErr := func(errInfo string) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(errInfo)); err != nil {
			klog.Errorf("failed write data to response. %v", err)
		}
	}

	options := &corev1.PodLogOptions{
		Container: c.containerName,
	}

	query := req.URL.Query()
	if lines := query.Get("tailLines"); lines != "" {
		numLine, err := strconv.ParseInt(lines, 10, 64)
		if err == nil {
			options.TailLines = &numLine
		}
	}
	if follow := query.Get("follow"); follow == "true" {
		options.Follow = true
	}
	if previous := query.Get("previous"); previous == "true" {
		options.Previous = true
	}
	if timestamps := query.Get("timestamps"); timestamps == "true" {
		options.Timestamps = true
	}
	if sinceSeconds := query.Get("sinceSeconds"); sinceSeconds != "" {
		seconds, err := strconv.ParseInt(sinceSeconds, 10, 64)
		if err == nil {
			options.SinceSeconds = &seconds
		}
	}

	logReq := c.logClient.CoreV1().Pods(c.podNamespace).GetLogs(c.podName, options)
	podlogs, err := logReq.Stream(context.Background())
	if err != nil {
		writeResponseErr(fmt.Sprintf("faield to stream log. %v", err))
		return
	}
	defer podlogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podlogs)
	if err != nil {
		writeResponseErr(fmt.Sprintf("faield to copy log. %v", err))
		return
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	_, err = w.Write(buf.Bytes())
	if err != nil {
		klog.Errorf("failed to write log to response. %v", err)
		return
	}
	return
}
