// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package server

import (
	"crypto/tls"
	"fmt"
	"net/http"

	restful "github.com/emicklei/go-restful"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/util/flushwriter"
	"k8s.io/klog"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/drivers"
)

// Server is a http.Handler which exposes kubelet functionality over HTTP.
type Server struct {
	auth          AuthInterface
	driverFactory *drivers.DriverFactory
	restfulCont   containerInterface
}

type TLSOptions struct {
	Config   *tls.Config
	CertFile string
	KeyFile  string
}

// AuthInterface contains all methods required by the auth filters
type AuthInterface interface {
	authenticator.Request
	authorizer.RequestAttributesGetter
	authorizer.Authorizer
}

// containerInterface defines the restful.Container functions used on the root container
type containerInterface interface {
	Add(service *restful.WebService) *restful.Container
	Handle(path string, handler http.Handler)
	Filter(filter restful.FilterFunction)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	RegisteredWebServices() []*restful.WebService

	// RegisteredHandlePaths returns the paths of handlers registered directly with the container (non-web-services)
	// Used to test filters are being applied on non-web-service handlers
	RegisteredHandlePaths() []string
}

// filteringContainer delegates all Handle(...) calls to Container.HandleWithFilter(...),
// so we can ensure restful.FilterFunctions are used for all handlers
type filteringContainer struct {
	*restful.Container
	registeredHandlePaths []string
}

func (a *filteringContainer) Handle(path string, handler http.Handler) {
	a.HandleWithFilter(path, handler)
	a.registeredHandlePaths = append(a.registeredHandlePaths, path)
}
func (a *filteringContainer) RegisteredHandlePaths() []string {
	return a.registeredHandlePaths
}

// NewServer initializes and configures a kubelet.Server object to handle HTTP requests.
func NewServer(
	driverFactory *drivers.DriverFactory,
	auth AuthInterface,
) Server {
	server := Server{
		driverFactory: driverFactory,
		auth:          auth,
		restfulCont:   &filteringContainer{Container: restful.NewContainer()},
	}
	if auth != nil {
		server.InstallAuthFilter()
	}
	server.InstallDefaultHandlers()
	return server
}

// InstallAuthFilter installs authentication filters with the restful Container.
func (s *Server) InstallAuthFilter() {
	s.restfulCont.Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		// Authenticate
		authResp, ok, err := s.auth.AuthenticateRequest(req.Request)
		if err != nil {
			klog.Errorf("unable to authenticate the request due to an error: %v", err)
			if err := resp.WriteErrorString(http.StatusUnauthorized, "Unauthorized"); err != nil {
				klog.Errorf("failed to write error to response, %v", err)
			}
			return
		}
		if !ok {
			klog.Errorf("unable to authenticate the request")
			if err := resp.WriteErrorString(http.StatusUnauthorized, "Unauthorized"); err != nil {
				klog.Errorf("failed to write error to response, %v", err)
			}
			return
		}

		// Get authorization attributes
		attrs := s.auth.GetRequestAttributes(authResp.User, req.Request)

		// Authorize
		decision, _, err := s.auth.Authorize(attrs)
		if err != nil {
			msg := fmt.Sprintf(
				"Authorization error (user=%s, verb=%s, resource=%s, subresource=%s)",
				attrs.GetUser().GetName(), attrs.GetVerb(), attrs.GetResource(), attrs.GetSubresource())
			klog.Error(msg)
			if err := resp.WriteErrorString(http.StatusInternalServerError, msg); err != nil {
				klog.Errorf("failed to write error to response, %v", err)
			}
			return
		}
		if decision != authorizer.DecisionAllow {
			msg := fmt.Sprintf(
				"Forbidden (user=%s, verb=%s, resource=%s, subresource=%s)",
				attrs.GetUser().GetName(), attrs.GetVerb(), attrs.GetResource(), attrs.GetSubresource())
			klog.Error(msg)
			if err := resp.WriteErrorString(http.StatusForbidden, msg); err != nil {
				klog.Errorf("failed to write error to response, %v", err)
			}
			return
		}

		// Continue
		chain.ProcessFilter(req, resp)
	})
}

// InstallDefaultHandlers registers the default set of supported HTTP request
// patterns with the restful Container.
func (s *Server) InstallDefaultHandlers() {
	// Log handler
	ws := new(restful.WebService)
	ws.
		Path("/containerLogs")
	ws.Route(ws.GET("/{podNamespace}/{podID}/{containerName}").
		To(s.getContainerLogs).
		Operation("getContainerLogs"))
	s.restfulCont.Add(ws)

	// Montior handler
	ws = new(restful.WebService)
	ws.
		Path("/monitoring/{queryPath}")
	ws.Route(ws.GET("").
		To(s.getMetrics).
		Operation("getMonitoringMetrics"))
	s.restfulCont.Add(ws)
}

// getContainerLogs handles containerLogs request against the klusterlet
func (s *Server) getContainerLogs(request *restful.Request, response *restful.Response) {
	podNamespace := request.PathParameter("podNamespace")
	podID := request.PathParameter("podID")
	containerName := request.PathParameter("containerName")
	query := request.Request.URL.Query()
	logger := s.driverFactory.LogDriver()

	if len(podID) == 0 {
		klog.Errorf("failed to get container logs due to missing podID")
		if err := response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Missing podID."}`)); err != nil {
			klog.Errorf("failed to write error to response, %v", err)
		}
		return
	}
	if len(containerName) == 0 {
		klog.Errorf("failed to get container logs due to missing container name")
		if err := response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Missing container name."}`)); err != nil {
			klog.Errorf("failed to write error to response, %v", err)
		}
		return
	}
	if len(podNamespace) == 0 {
		klog.Errorf("failed to get container logs due to missing podNamespace")
		if err := response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Missing podNamespace."}`)); err != nil {
			klog.Errorf("failed to write error to response, %v", err)
		}
		return
	}

	fw := flushwriter.Wrap(response.ResponseWriter)
	response.Header().Set("Transfer-Encoding", "chunked")
	if err := logger.GetContainerLog(podNamespace, podID, containerName, query, fw); err != nil {
		klog.Errorf("failed to get container logs, %v", err)
		if err := response.WriteError(http.StatusBadRequest, err); err != nil {
			klog.Errorf("failed to write error to response, %v", err)
		}
		return
	}
}

// getMetrics handles monitoring metrics against the klusterlet
func (s *Server) getMetrics(request *restful.Request, response *restful.Response) {
	monitor := s.driverFactory.MonitorDriver()
	queryPath := request.PathParameter("queryPath")
	query := request.Request.URL.Query()

	fw := flushwriter.Wrap(response.ResponseWriter)
	response.Header().Set("Transfer-Encoding", "chunked")
	if err := monitor.GetMetrics(queryPath, query, fw); err != nil {
		klog.Errorf("failed to get monitor data: %v", err)
		if err := response.WriteError(http.StatusBadRequest, err); err != nil {
			klog.Errorf("failed to write error to response, %v", err)
		}
		return
	}
}

// ServeHTTP responds to HTTP requests on the Kubelet.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.restfulCont.ServeHTTP(w, req)
}
