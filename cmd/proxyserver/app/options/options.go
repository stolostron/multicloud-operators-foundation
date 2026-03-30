// Copyright (c) 2020 Red Hat, Inc.

package options

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/stolostron/cluster-lifecycle-api/helpers/tlsprofile"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/api"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/openapi"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericapiserveroptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/rest"
)

type Options struct {
	KubeConfigFile  string
	ConfigMapLabels string
	ServerRun       *genericapiserveroptions.ServerRunOptions
	SecureServing   *genericapiserveroptions.SecureServingOptionsWithLoopback
	Authentication  *genericapiserveroptions.DelegatingAuthenticationOptions
	Authorization   *genericapiserveroptions.DelegatingAuthorizationOptions
	ClientOptions   *ClientOptions
}

// NewOptions constructs a new set of default options for proxyserver.
func NewOptions() *Options {
	return &Options{
		KubeConfigFile:  "",
		ConfigMapLabels: "config=acm-proxyserver",
		ServerRun:       genericapiserveroptions.NewServerRunOptions(),
		SecureServing:   genericapiserveroptions.NewSecureServingOptions().WithLoopback(),
		Authentication:  genericapiserveroptions.NewDelegatingAuthenticationOptions(),
		Authorization:   genericapiserveroptions.NewDelegatingAuthorizationOptions(),
		ClientOptions:   NewClientOptions(),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.KubeConfigFile, "kube-config-file", o.KubeConfigFile, "Kubernetes configuration file to connect to kube-apiserver")
	fs.StringVar(&o.ConfigMapLabels, "configmap-labels", o.ConfigMapLabels, "Labels of configMap. Default is config=acm-proxyserver")
	o.ServerRun.AddUniversalFlags(fs)
	o.SecureServing.AddFlags(fs)
	o.Authentication.AddFlags(fs)
	o.Authorization.AddFlags(fs)
	o.ClientOptions.AddFlags(fs)
}

func (o *Options) SetDefaults() error {
	if err := o.ServerRun.DefaultAdvertiseAddress(o.SecureServing.SecureServingOptions); err != nil {
		return err
	}

	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts(o.ServerRun.AdvertiseAddress.String(), nil, nil); err != nil {
		return fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	return nil
}

// APIServerConfig creates the generic API server configuration with OpenShift TLS profile support.
// It reads the TLS profile from OpenShift APIServer CR and applies the appropriate TLS version
// and cipher suites to the secure serving configuration. On non-OpenShift clusters, defaults to TLS 1.2.
func (o *Options) APIServerConfig(kubeConfig *rest.Config) (*genericapiserver.Config, error) {
	serverConfig := genericapiserver.NewConfig(api.Codecs)
	if err := o.ServerRun.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	if err := o.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	// Get TLS profile from OpenShift APIServer CR
	// Returns default TLS 1.2 config on non-OpenShift clusters (NotFound handled internally)
	// Returns error only for real configuration problems
	tlsConfig, err := tlsprofile.GetTLSConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get TLS configuration: %w", err)
	}

	// Apply TLS settings to generic API server's SecureServingInfo
	serverConfig.SecureServing.MinTLSVersion = tlsConfig.MinVersion
	if len(tlsConfig.CipherSuites) > 0 {
		serverConfig.SecureServing.CipherSuites = tlsConfig.CipherSuites
	}

	if err := o.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, nil); err != nil {
		return nil, err
	}

	// TODO: add custormer authorization here
	if err := o.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
		return nil, err
	}

	// enable OpenAPI schemas
	namer := openapinamer.NewDefinitionNamer(api.Scheme)

	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(
		openapi.GetOpenAPIDefinitions, namer)
	serverConfig.OpenAPIConfig.Info.Title = "foundation-proxy-server"
	serverConfig.OpenAPIConfig.Info.Version = "0.0.1"

	// related issue https://github.com/kubernetes/kubernetes/pull/114998 brought from apiserver upgrade to 0.28.3
	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(openapi.GetOpenAPIDefinitions, namer)
	serverConfig.OpenAPIV3Config.Info.Title = "foundation-proxy-server"
	serverConfig.OpenAPIV3Config.Info.Version = "0.0.1"
	return serverConfig, nil
}

func (o *Options) Validate() []error {
	var errors []error
	if errs := o.ServerRun.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if errs := o.SecureServing.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if errs := o.Authentication.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	if errs := o.Authorization.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// TODO: add more checks
	return errors
}
