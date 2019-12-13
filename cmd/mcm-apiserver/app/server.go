// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package app

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apiserver/authorization"

	"github.com/go-openapi/spec"
	"github.com/spf13/cobra"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"

	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/mcm-apiserver/app/options"
	api "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/api"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/openapi"
	hcmv1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apiserver"
	hadmission "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apiserver/admission"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apiserver/authenticator"
	apiserveroptions "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apiserver/options"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/internalclientset"
	clusterclient "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset"
	clusterinformers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	informers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated/internalversion"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/version"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/admission"
	admissionmetrics "k8s.io/apiserver/pkg/admission/metrics"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/filters"
	serveroptions "k8s.io/apiserver/pkg/server/options"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// NewAPIServerCommand creates a *cobra.Command object with default parameters
func NewAPIServerCommand() *cobra.Command {
	s := options.NewServerRunOptions()
	cmd := &cobra.Command{
		Use:  "apiserver",
		Long: ``,
		Run: func(cmd *cobra.Command, args []string) {
			if err := Run(s, wait.NeverStop); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	s.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the specified APIServer.  It only returns if stopCh is closed
// or one of the ports cannot be listened on initially.
func Run(s *options.ServerRunOptions, stopCh <-chan struct{}) error {
	err := NonBlockingRun(s, stopCh)
	if err != nil {
		return err
	}
	<-stopCh
	return nil
}

// AddSniffFilterHandlerChain add X-Content-Type-Options header, THIS SHOULD BE REMOVED AFTER WE UPDATE Kube API
func AddSniffFilterHandlerChain(apiHandler http.Handler, c *genericapiserver.Config) http.Handler {
	handler := genericapiserver.DefaultBuildHandlerChain(apiHandler, c)
	handler = withSniffFilterHandler(handler)
	handler = withCacheControlHandler(handler)

	return handler
}

func withSniffFilterHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasContentType := w.Header().Get("Content-Type")
		if len(hasContentType) == 0 {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		handler.ServeHTTP(w, r)
	})
}

func withCacheControlHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		handler.ServeHTTP(w, r)
	})
}

// NonBlockingRun runs the specified APIServer and configures it to
// stop with the given channel.
func NonBlockingRun(s *options.ServerRunOptions, stopCh <-chan struct{}) error {
	// set defaults
	if err := s.GenericServerRunOptions.DefaultAdvertiseAddress(s.SecureServing.SecureServingOptions); err != nil {
		return err
	}
	if err := s.SecureServing.MaybeDefaultWithSelfSignedCerts(s.GenericServerRunOptions.AdvertiseAddress.String(), nil, nil); err != nil {
		return fmt.Errorf("error creating self-signed certificates: %v", err)
	}
	if err := s.KlusterletClientOptions.MaybeDefaultWithSelfSignedCerts(s.GenericServerRunOptions.AdvertiseAddress.String()); err != nil {
		return fmt.Errorf("error creating self-signed klusterlet certificates: %v", err)
	}

	// validate options
	if errs := s.Validate(); len(errs) != 0 {
		return utilerrors.NewAggregate(errs)
	}

	genericConfig := genericapiserver.NewConfig(api.Codecs)

	genericConfig.BuildHandlerChainFunc = AddSniffFilterHandlerChain

	if err := s.GenericServerRunOptions.ApplyTo(genericConfig); err != nil {
		return err
	}
	if err := s.SecureServing.ApplyTo(&genericConfig.SecureServing, &genericConfig.LoopbackClientConfig); err != nil {
		return err
	}

	hcmVersion := version.Get()
	genericConfig.Version = &hcmVersion
	klusterletClientConfig := s.KlusterletClientOptions.Config()

	var sharedInformers informers.SharedInformerFactory
	var kubeSharedInformers kubeinformers.SharedInformerFactory
	var clusterInformers clusterinformers.SharedInformerFactory
	if !s.StandAlone {
		internalClient, kubeClient, clusterClient, err := generateClients(genericConfig)
		if err != nil {
			return err
		}
		sharedInformers = informers.NewSharedInformerFactory(internalClient, 10*time.Minute)
		kubeSharedInformers = kubeinformers.NewSharedInformerFactory(kubeClient, 10*time.Minute)
		clusterInformers = clusterinformers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
		genericConfig.AdmissionControl, err = buildAdmission(
			s, internalClient, sharedInformers, kubeClient, kubeSharedInformers, clusterClient, clusterInformers)
		if err != nil {
			return fmt.Errorf("failed to initialize admission: %v", err)
		}

		if err := s.Audit.ApplyTo(
			genericConfig,
			genericConfig.LoopbackClientConfig,
			kubeSharedInformers,
			serveroptions.NewProcessInfo("mcm-apiserver", "kube-system"),
			&serveroptions.WebhookOptions{},
		); err != nil {
			return err
		}
		klusterletClientConfig.KubeClient = kubeClient
	}

	genericConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(openapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(api.Scheme))
	if genericConfig.OpenAPIConfig.Info == nil {
		genericConfig.OpenAPIConfig.Info = &spec.Info{}
	}
	if genericConfig.OpenAPIConfig.Info.Version == "" {
		if genericConfig.Version != nil {
			genericConfig.OpenAPIConfig.Info.Version = strings.Split(genericConfig.Version.String(), "-")[0]
		} else {
			genericConfig.OpenAPIConfig.Info.Version = "unversioned"
		}
	}

	if !s.StandAlone {
		if err := s.Authentication.ApplyTo(
			&genericConfig.Authentication,
			genericConfig.SecureServing,
			genericConfig.OpenAPIConfig); err != nil {
			return err
		}

		// hcmAuthorization is used to refactor DelegatingAuthorizationOptions of apiserver authorization to set qps/burst.
		// if apiserver can support to set qps/burst of authorization request, need to modify back.
		hcmAuthorization := authorization.HcmDelegatingAuthorizationOptions{
			DelegatingAuthorizationOptions: s.Authorization,
			QPS:                            s.AuthorizationQPS,
			Burst:                          s.AuthorizationBurst,
		}
		if err := hcmAuthorization.ApplyTo(&genericConfig.Authorization); err != nil {
			return err
		}
	} else {
		// always warn when auth is disabled, since this should only be used for testing
		klog.Warning("Authentication and authorization disabled for testing purposes")
		genericConfig.Authentication.Authenticator = &authenticator.AnyUserAuthenticator{}
		genericConfig.Authorization.Authorizer = authorizerfactory.NewAlwaysAllowAuthorizer()
	}

	resourceConfig := defaultResourceConfig()
	if err := s.APIEnablement.ApplyTo(genericConfig, resourceConfig, api.Scheme); err != nil {
		return err
	}

	// The API server stores objects using a particular API version for each
	// group, regardless of API version of the object when it was created.
	//
	// storageGroupsToEncodingVersion holds a map of API group to version that
	// the API server uses to store that group.
	storageGroupsToEncodingVersion, err := apiserveroptions.NewStorageSerializationOptions().StorageGroupsToEncodingVersion()
	if err != nil {
		return fmt.Errorf("error generating storage version map: %s", err)
	}

	// Build the default storage factory.
	//
	// The default storage factory returns the storage interface for a
	// particular GroupResource (an (api-group, resource) tuple).
	storageFactory, err := apiserver.NewStorageFactory(
		s.Etcd.StorageConfig,
		s.Etcd.DefaultStorageMediaType,
		api.Codecs,
		serverstorage.NewDefaultResourceEncodingConfig(api.Scheme),
		storageGroupsToEncodingVersion,
		nil, /* group storage version overrides */
		apiserver.DefaultAPIResourceConfigSource(),
		nil, /* resource config overrides */
	)
	if err != nil {
		return err
	}

	if err = s.Etcd.ApplyWithStorageFactoryTo(storageFactory, genericConfig); err != nil {
		return err
	}

	genericConfig.SwaggerConfig = genericapiserver.DefaultSwaggerConfig()
	genericConfig.LongRunningFunc = filters.BasicLongRunningRequestCheck(
		sets.NewString("watch", "proxy"),
		sets.NewString("attach", "exec", "proxy", "log", "portforward"),
	)

	m, err := genericConfig.Complete(nil).New("hcm", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return err
	}

	apiResourceConfigSource := storageFactory.APIResourceConfigSource
	installHCMAPIs(
		m, genericConfig.RESTOptionsGetter, apiResourceConfigSource,
		s.MCMStorage, klusterletClientConfig)

	if !s.StandAlone {
		sharedInformers.Start(stopCh)
		kubeSharedInformers.Start(stopCh)
		clusterInformers.Start(stopCh)
	}

	err = m.PrepareRun().NonBlockingRun(stopCh)
	return err
}

func defaultResourceConfig() *serverstorage.ResourceConfig {
	rc := serverstorage.NewResourceConfig()

	rc.EnableVersions(
		hcmv1alpha1.SchemeGroupVersion,
		clusterv1alpha1.SchemeGroupVersion,
	)

	return rc
}

// buildAdmission constructs the admission chain
func buildAdmission(s *options.ServerRunOptions,
	client internalclientset.Interface, sharedInformers informers.SharedInformerFactory,
	kubeClient kubeclientset.Interface, kubeSharedInformers kubeinformers.SharedInformerFactory,
	clusterClient clusterclient.Interface, clusterInformers clusterinformers.SharedInformerFactory) (admission.Interface, error) {
	admissionControlPluginNames := s.Admission.EnablePlugins
	klog.Infof("Admission control plugin names: %v", admissionControlPluginNames)
	var err error

	pluginInitializer := hadmission.NewPluginInitializer(
		client, sharedInformers, kubeClient, kubeSharedInformers, clusterClient, clusterInformers)
	admissionConfigProvider, err := admission.ReadAdmissionConfiguration(
		admissionControlPluginNames, s.Admission.ConfigFile, api.Scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin config: %v", err)
	}
	return s.Admission.Plugins.NewFromPlugins(
		admissionControlPluginNames,
		admissionConfigProvider,
		pluginInitializer,
		admission.DecoratorFunc(admissionmetrics.WithControllerMetrics))
}

func generateClients(svrConfig *genericapiserver.Config) (
	*internalclientset.Clientset, *kubeclientset.Clientset, *clusterclient.Clientset, error) {
	internalClient, err := internalclientset.NewForConfig(svrConfig.LoopbackClientConfig)
	if err != nil {
		klog.Errorf("Failed to create clientset for hcm self-communication: %v", err)
		return nil, nil, nil, err
	}
	inClusterConfig, err := restclient.InClusterConfig()
	if err != nil {
		klog.Errorf("Failed to get kube client config: %v", err)
		return nil, nil, nil, err
	}
	inClusterConfig.GroupVersion = &schema.GroupVersion{}

	kubeClient, err := kubeclientset.NewForConfig(inClusterConfig)
	if err != nil {
		klog.Errorf("Failed to create clientset interface: %v", err)
		return nil, nil, nil, err
	}

	clusterClient, err := clusterclient.NewForConfig(inClusterConfig)
	if err != nil {
		klog.Errorf("Failed to create cluster client: %v", err)
		return nil, nil, nil, err
	}

	return internalClient, kubeClient, clusterClient, nil
}
