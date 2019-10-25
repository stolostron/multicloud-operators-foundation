// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package server

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/klusterlet/drivers"
)

type fakeAuth struct {
	authenticateFunc func(*http.Request) (*authenticator.Response, bool, error)
	attributesFunc   func(user.Info, *http.Request) authorizer.Attributes
	authorizeFunc    func(authorizer.Attributes) (authorized authorizer.Decision, reason string, err error)
}

func (f *fakeAuth) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	return f.authenticateFunc(req)
}
func (f *fakeAuth) GetRequestAttributes(u user.Info, req *http.Request) authorizer.Attributes {
	return f.attributesFunc(u, req)
}
func (f *fakeAuth) Authorize(a authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	return f.authorizeFunc(a)
}

type fakeDriver struct {
	content string
}

func (f *fakeDriver) GetContainerLog(namespace, podID, containerName string, query url.Values, stdout io.Writer) error {
	_, err := stdout.Write([]byte(f.content))
	return err
}

func (f *fakeDriver) GetMetrics(queryPath string, query url.Values, stdout io.Writer) error {
	_, err := stdout.Write([]byte(f.content))
	return err
}

func (f *fakeDriver) SetContent(content string) {
	f.content = content
}

type serverTestFramework struct {
	serverUnderTest *Server
	fakeAuth        *fakeAuth
	fakeDriver      *fakeDriver
	testHTTPServer  *httptest.Server
}

func newServerTest() *serverTestFramework {
	fw := &serverTestFramework{}
	fw.fakeAuth = &fakeAuth{
		authenticateFunc: func(req *http.Request) (*authenticator.Response, bool, error) {
			return &authenticator.Response{User: &user.DefaultInfo{Name: "test"}}, true, nil
		},
		attributesFunc: func(u user.Info, req *http.Request) authorizer.Attributes {
			return &authorizer.AttributesRecord{User: u}
		},
		authorizeFunc: func(a authorizer.Attributes) (decision authorizer.Decision, reason string, err error) {
			return authorizer.DecisionAllow, "", nil
		},
	}

	fw.fakeDriver = &fakeDriver{}
	factory := &drivers.DriverFactory{}
	factory.SetLogDriver(fw.fakeDriver)
	factory.SetMonitorDriver(fw.fakeDriver)
	server := NewServer(factory, fw.fakeAuth)
	fw.serverUnderTest = &server
	fw.testHTTPServer = httptest.NewServer(fw.serverUnderTest)

	return fw
}

func TestContainerLogs(t *testing.T) {
	fw := newServerTest()
	defer fw.testHTTPServer.Close()

	output := "foo bar"

	fw.fakeDriver.SetContent(output)

	resp, err := http.Get(fw.testHTTPServer.URL + "/containerLogs/default/pod1/container1")
	if err != nil {
		t.Errorf("Got error GETing: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading container logs: %v", err)
	}

	result := string(body)
	if result != output {
		t.Errorf("Expected: '%v', got: '%v'", output, result)
	}
}

func TestMetrics(t *testing.T) {
	fw := newServerTest()
	defer fw.testHTTPServer.Close()

	output := "foo bar"

	fw.fakeDriver.SetContent(output)

	resp, err := http.Get(fw.testHTTPServer.URL + "/monitoring/federate")
	if err != nil {
		t.Errorf("Got error GETing: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading container logs: %v", err)
	}

	result := string(body)
	if result != output {
		t.Errorf("Expected: '%v', got: '%v'", output, result)
	}
}
