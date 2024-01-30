// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"context"
	"k8s.io/klog"
	"net"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// ServeHealthProbes starts a server to check healthz and readyz probes
func ServeHealthProbes(stop <-chan struct{}, healthProbeBindAddress string, configCheck healthz.Checker) {
	healthzHandler := &healthz.Handler{Checks: map[string]healthz.Checker{
		"healthz-ping": healthz.Ping,
		"configz-ping": configCheck,
	}}
	readyzHandler := &healthz.Handler{Checks: map[string]healthz.Checker{
		"readyz-ping": healthz.Ping,
	}}

	mux := http.NewServeMux()
	mux.Handle("/readyz", http.StripPrefix("/readyz", readyzHandler))
	mux.Handle("/healthz", http.StripPrefix("/healthz", healthzHandler))

	server := http.Server{
		Handler: mux,
	}

	ln, err := net.Listen("tcp", healthProbeBindAddress)
	if err != nil {
		klog.Errorf("error listening on %s: %v", ":8000", err)
		return
	}

	klog.Infof("heath probes server is running...")
	// Run server
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			klog.Fatal(err)
		}
	}()

	// Shutdown the server when stop is closed
	<-stop
	if err := server.Shutdown(context.Background()); err != nil {
		klog.Fatal(err)
	}
}
