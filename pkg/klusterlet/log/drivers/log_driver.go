package drivers

import (
	"context"
	"io"
	"net/url"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

type LogDriver struct {
	kubeclient kubernetes.Interface
}

func NewLogDriver(kubeclient kubernetes.Interface) *LogDriver {
	return &LogDriver{
		kubeclient: kubeclient,
	}
}

func (l *LogDriver) GetContainerLog(namespace, podID, containerName string, query url.Values, stdout io.Writer) error {
	options := &corev1.PodLogOptions{
		Container: containerName,
	}

	if lines := query.Get("tailLines"); lines != "" {
		numline, err := strconv.ParseInt(lines, 10, 64)
		if err == nil {
			options.TailLines = &numline
		}
	}
	if follow := query.Get("follow"); follow != "" && follow == "true" {
		options.Follow = true
	}
	if previous := query.Get("previous"); previous != "" && previous == "true" {
		options.Previous = true
	}
	if timestamps := query.Get("timestamps"); timestamps != "" && timestamps == "true" {
		options.Timestamps = true
	}
	if sinceSeconds := query.Get("sinceSeconds"); sinceSeconds != "" {
		seconds, err := strconv.ParseInt(sinceSeconds, 10, 64)
		if err == nil {
			options.SinceSeconds = &seconds
		}
	}

	request := l.kubeclient.CoreV1().Pods(namespace).GetLogs(podID, options)
	readCloser, err := request.Stream(context.TODO())
	if err != nil {
		klog.Errorf("Failed to read logs %v", err)
		return err
	}
	defer readCloser.Close()

	_, err = io.Copy(stdout, readCloser)
	if err != nil {
		klog.Errorf("Failed to copy logs to writer %v", err)
		return err
	}

	return nil
}
