package options

import (
	"fmt"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/api"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/apis/openapi"
	"github.com/spf13/pflag"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericapiserveroptions "k8s.io/apiserver/pkg/server/options"
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

	if err := o.ClientOptions.MaybeDefaultWithSelfSignedCerts(o.ServerRun.AdvertiseAddress.String()); err != nil {
		return fmt.Errorf("error creating self-signed certificates: %v", err)
	}
	return nil
}

func (o *Options) APIServerConfig() (*genericapiserver.Config, error) {
	serverConfig := genericapiserver.NewConfig(api.Codecs)
	if err := o.ServerRun.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	if err := o.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	if err := o.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, nil); err != nil {
		return nil, err
	}

	//TODO: add custormer authorization here
	if err := o.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
		return nil, err
	}

	// enable OpenAPI schemas
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(
		openapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(api.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "acm-proxy-server"
	serverConfig.OpenAPIConfig.Info.Version = "0.0.1"

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
