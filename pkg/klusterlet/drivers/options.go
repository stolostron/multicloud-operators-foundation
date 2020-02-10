// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package drivers

import (
	kubelogdriver "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/drivers/kube"
	prometheus "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/drivers/prometheus"
	"github.com/spf13/pflag"
)

type DriverFactoryOptions struct {
	LogDriverType            string
	MonitorDriverType        string
	PrometheusService        string
	PrometheusSecret         string
	PrometheusUseBearerToken bool
	PrometheusScrapeTarget   string
}

func NewDriverFactoryOptions() *DriverFactoryOptions {
	return &DriverFactoryOptions{
		LogDriverType:            kubelogdriver.DriverType,
		MonitorDriverType:        prometheus.NativeDriverType,
		PrometheusService:        "",
		PrometheusSecret:         "",
		PrometheusUseBearerToken: false,
		PrometheusScrapeTarget:   "kubernetes-cadvisor",
	}
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (d *DriverFactoryOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&d.LogDriverType, "log-driver", d.LogDriverType, ""+
		"Driver to get log, default uses kube-apiserver")
	fs.StringVar(&d.MonitorDriverType, "monitor-driver", d.MonitorDriverType, ""+
		"Driver for monitor, default uses native prometheus")
	fs.StringVar(&d.PrometheusService, "prometheus-service", d.PrometheusService, ""+
		"Prometheus service in managed cluster, in the format of namespace/name")
	fs.StringVar(&d.PrometheusSecret, "prometheus-secret", d.PrometheusSecret, ""+
		"Secrets to visit prometheus in managed cluster, in the format of namespace/name")
	fs.BoolVar(&d.PrometheusUseBearerToken, "prometheus-use-bearer-token", d.PrometheusUseBearerToken, ""+
		"use bearer token for prometheus in Openshift")
	fs.StringVar(&d.PrometheusScrapeTarget, "prometheus-scrape-target", d.PrometheusScrapeTarget, ""+
		"The job name assigned to scraped metrics")
}
