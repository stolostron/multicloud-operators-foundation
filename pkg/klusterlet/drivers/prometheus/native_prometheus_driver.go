// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package prometheus

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type NativePrometheusDriver struct {
	address        string
	httpClient     *http.Client
	port           int32
	useBearerToken bool
}

const (
	keyFile          = "tls.key"
	certFile         = "tls.crt"
	bearerTokenPath  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	NativeDriverType = "native-prometheus"
)

func NewNativePrometheusDriver(
	kubeclient kubernetes.Interface, prometheusService, prometheusSecret string, useBearerToken bool) (*NativePrometheusDriver, error) {
	promNamespace, promName, err := cache.SplitMetaNamespaceKey(prometheusService)
	if err != nil {
		return nil, err
	}

	promService, err := kubeclient.CoreV1().Services(promNamespace).Get(promName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if len(promService.Spec.Ports) == 0 {
		return nil, fmt.Errorf("port should have value")
	}
	promPort := promService.Spec.Ports[0].Port
	config := &tls.Config{}
	config.InsecureSkipVerify = true

	// The kubernetes package [cache] method [SplitMetaNamespaceKey] has a bug now
	// The bug is when input string is "" (empty), the method [SplitMetaNamespaceKey] will return "" "" nil
	// which is not expected.
	// When the input string is "" (empty), the method should return "" "" nil
	// So here, we use secretName != "" to check to use TLS Secret or not, rather than using err == nil
	secretNamespace, secretName, err := cache.SplitMetaNamespaceKey(prometheusSecret)
	if err != nil {
		return nil, err
	}

	if secretName != "" {
		// use TLS Secret
		promSecret, err := kubeclient.CoreV1().Secrets(secretNamespace).Get(secretName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		certificate, err := tls.X509KeyPair(promSecret.Data[certFile], promSecret.Data[keyFile])
		if err != nil {
			return nil, err
		}
		config.Certificates = []tls.Certificate{certificate}
		config.BuildNameToCertificate()
	}

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: config}}
	return &NativePrometheusDriver{
		address:        promService.Spec.ClusterIP,
		port:           promPort,
		httpClient:     client,
		useBearerToken: useBearerToken,
	}, nil
}

func (p *NativePrometheusDriver) GetMetrics(queryPath string, query url.Values, stdout io.Writer) error {
	path := fmt.Sprintf("%s:%d/%s", p.address, p.port, queryPath)

	// always use https
	path = fmt.Sprintf("https://%s", path)

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return err
	}

	if p.useBearerToken {
		token, tokenErr := ioutil.ReadFile(bearerTokenPath)
		if tokenErr != nil {
			return tokenErr
		}

		bearer := "Bearer " + string(token)
		req.Header.Add("Authorization", bearer)
	}

	req.URL.RawQuery = query.Encode()
	resp, err := p.httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	_, err = io.Copy(stdout, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
