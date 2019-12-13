// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// IBM Confidential
// OCO Source Materials
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been
// deposited with the U.S. Copyright Office.

package helm

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/version"
)

type InSecureHTTPGetter struct { //nolint
	client *http.Client
}

//SetCredentials sets the credentials for the getter
func (g *InSecureHTTPGetter) SetCredentials(username, password string) {
}

//Get performs a Get from repo.Getter and returns the body.
func (g *InSecureHTTPGetter) Get(href string) (*bytes.Buffer, error) {
	return g.get(href)
}

func (g *InSecureHTTPGetter) get(href string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)

	// Set a helm specific user agent so that a repo server and metrics can
	// separate helm calls from other tools interacting with repos.
	req, err := http.NewRequest("GET", href, nil)
	if err != nil {
		return buf, err
	}
	req.Header.Set("User-Agent", "Helm/"+strings.TrimPrefix(version.GetVersion(), "v"))

	resp, err := g.client.Do(req)
	if err != nil {
		return buf, err
	}
	if resp.StatusCode != 200 {
		return buf, fmt.Errorf("failed to fetch %s : %s", href, resp.Status)
	}

	_, err = io.Copy(buf, resp.Body)
	resp.Body.Close()
	return buf, err
}

func newInSecureHTTPGetter(url, certFile, keyFile, caFile string) (getter.Getter, error) {
	return NewInSecureHTTPGetter(url, certFile, keyFile, caFile)
}

// NewInSecureHTTPGetter creates an insecure HTTP getter
func NewInSecureHTTPGetter(url, certFile, keyFile, caFile string) (*InSecureHTTPGetter, error) {
	var client InSecureHTTPGetter
	tr := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client.client = &http.Client{Transport: tr}
	return &client, nil
}

// NewInSecureProviders create providers for mcm hub cluster helm repo
func NewInSecureProviders() getter.Providers {
	result := getter.Providers{
		{
			Schemes: []string{"https"},
			New:     newInSecureHTTPGetter,
		},
	}

	return result
}
