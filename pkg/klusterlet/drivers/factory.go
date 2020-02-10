// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package drivers

import (
	kubelogdriver "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/drivers/kube"
	prometheus "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/drivers/prometheus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// DriverFactory is factory to install all drivers
type DriverFactory struct {
	logDriver     LogInterface
	monitorDriver MonitorInterface
}

func NewDriverFactory(options *DriverFactoryOptions, kubeclient kubernetes.Interface) *DriverFactory {
	factory := &DriverFactory{}
	if options.LogDriverType == kubelogdriver.DriverType {
		factory.logDriver = kubelogdriver.NewLogDriver(kubeclient)
	}

	if options.MonitorDriverType == prometheus.NativeDriverType {
		driver, err := prometheus.NewNativePrometheusDriver(
			kubeclient, options.PrometheusService, options.PrometheusSecret, options.PrometheusUseBearerToken)
		if err != nil {
			klog.Errorf("Failed to start monitor driver: %v", err)
		}
		factory.monitorDriver = driver
	}

	return factory
}

func (d *DriverFactory) LogDriver() LogInterface {
	return d.logDriver
}

func (d *DriverFactory) SetLogDriver(logDriver LogInterface) {
	d.logDriver = logDriver
}

func (d *DriverFactory) MonitorDriver() MonitorInterface {
	return d.monitorDriver
}

func (d *DriverFactory) SetMonitorDriver(monitorDriver MonitorInterface) {
	d.monitorDriver = monitorDriver
}
