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

package app

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/klusterlet/app/options"
	clientset "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	clusterclientset "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset"
	informers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/klusterlet"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/klusterlet/drivers"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/klusterlet/server"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/signals"
	helmutil "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/helm"
	restutils "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/rest"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apilabels "k8s.io/apimachinery/pkg/labels"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"

	routev1 "github.com/openshift/client-go/route/clientset/versioned"
)

const (
	componentKlusterlet = "klusterlet"
	prefixLabel         = "mcm.ibm.com/deployer-prefix"
)

// NewKlusterletCommand creates a *cobra.Command object with default parameters
func NewKlusterletCommand() *cobra.Command {
	s := options.NewKlusterletRunOptions()
	cmd := &cobra.Command{
		Use:  componentKlusterlet,
		Long: ``,
		Run: func(cmd *cobra.Command, args []string) {
			stopCh := signals.SetupSignalHandler()
			if err := Run(s, stopCh); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	s.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the specified klusterlet.  It only returns if stopCh is closed
// or one of the ports cannot be listened on initially.
func Run(s *options.KlusterletRunOptions, stopCh <-chan struct{}) error {
	err := RunKlusterletServer(s, stopCh)
	if err != nil {
		return err
	}
	<-stopCh
	return nil
}

// InitializeTLS checks for a configured TLSCertFile and TLSPrivateKeyFile: if unspecified a new self-signed
// certificate and key file are generated. Returns a configured server.TLSOptions object.
func InitializeTLS(s *options.KlusterletRunOptions) (*server.TLSOptions, error) {
	if s.TLSCertFile == "" && s.TLSPrivateKeyFile == "" {
		s.TLSCertFile = path.Join(s.CertDir, "kubelet.crt")
		s.TLSPrivateKeyFile = path.Join(s.CertDir, "kubelet.key")

		canReadCertAndKey, err := certutil.CanReadCertAndKey(s.TLSCertFile, s.TLSPrivateKeyFile)
		if err != nil {
			return nil, err
		}
		if !canReadCertAndKey {
			cert, key, err := certutil.GenerateSelfSignedCertKey(s.KlusterletAddress, nil, nil)
			if err != nil {
				return nil, fmt.Errorf("unable to generate self signed cert: %v", err)
			}

			if err := certutil.WriteCert(s.TLSCertFile, cert); err != nil {
				return nil, err
			}

			if err := certutil.WriteKey(s.TLSPrivateKeyFile, key); err != nil {
				return nil, err
			}

			klog.V(4).Infof("Using self-signed cert (%s, %s)", s.TLSCertFile, s.TLSPrivateKeyFile)
		}
	}

	tlsOptions := &server.TLSOptions{
		CertFile: s.TLSCertFile,
		KeyFile:  s.TLSPrivateKeyFile,
		Config:   &tls.Config{},
	}

	if len(s.ClientCAFile) > 0 {
		clientCAs, err := certutil.NewPool(s.ClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("unable to load client CA file %s: %v", s.ClientCAFile, err)
		}
		// Specify allowed CAs for client certificates
		tlsOptions.Config.ClientCAs = clientCAs
		// Populate PeerCertificates in requests, but don't reject connections without verified certificates
		tlsOptions.Config.ClientAuth = tls.RequestClientCert
	}

	return tlsOptions, nil
}

// RunKlusterletServer start a Klusterlet server
func RunKlusterletServer(s *options.KlusterletRunOptions, stopCh <-chan struct{}) error {
	// Validate clusterName and clusterNamespace
	convertedClusterName := strings.ToLower(strings.Replace(s.ClusterName, ".", "-", -1))
	if msg := validation.NameIsDNS1035Label(convertedClusterName, false); len(msg) != 0 {
		klog.Fatalf("Cluster name format error: %s", strings.Join(msg, ","))
	}
	if msg := validation.ValidateNamespaceName(s.ClusterNamespace, false); len(msg) != 0 {
		klog.Fatalf("Cluster namespace format error: %s", strings.Join(msg, ","))
	}

	clusterCfg, err := clientcmd.BuildConfigFromFlags("", s.ClusterConfigFile)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	clusterClient, err := kubernetes.NewForConfig(clusterCfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	kubeDynamicClientSet, err := dynamic.NewForConfig(clusterCfg)
	if err != nil {
		klog.Fatalf("Error building hcm kubernetes Dynamic clientset: %s", err.Error())
	}

	serverVersion, err := clusterClient.ServerVersion()
	if err != nil {
		klog.Fatalf("Failed to connect to kubernetes cluster: %s", err.Error())
	}
	klog.Info("Successful initial request to the kubernetes")

	var isOpenShift = false
	serverGroups, err := clusterClient.ServerGroups()
	if err != nil {
		klog.Fatalf("Failed to get server groups: %s", err.Error())
	}
	for _, apiGroup := range serverGroups.Groups {
		if apiGroup.Name == "project.openshift.io" {
			isOpenShift = true
			break
		}
	}

	var isAKS = false
	_, err = clusterClient.CoreV1().ServiceAccounts("kube-system").Get("omsagent", metav1.GetOptions{})
	if err == nil {
		isAKS = true
	}

	controllerCfg, err := clientcmd.BuildConfigFromFlags("", s.ControllerConfigFile)
	if err != nil {
		klog.Fatalf("Error building klusterlet config: %s", err.Error())
	}

	controllerClient, err := clientset.NewForConfig(controllerCfg)
	if err != nil {
		klog.Fatalf("Error building hcm clientset: %s", err.Error())
	}

	controllerKubeClient, err := kubernetes.NewForConfig(controllerCfg)
	if err != nil {
		klog.Fatalf("Error building hcm kubernetes clientset: %s", err.Error())
	}

	_, err = controllerClient.ServerVersion()
	if err != nil {
		klog.Fatalf("Failed to connect to hcm apiserver: %s", err.Error())
	}
	klog.Info("Successful initial request to the hcm apiserver")

	clusterClientSet, err := clusterclientset.NewForConfig(controllerCfg)
	if err != nil {
		klog.Fatalf("Error building cluster registry clientset: %s", err.Error())
	}

	tlsOptions, err := InitializeTLS(s)
	if err != nil {
		klog.Fatalln("Failed to initialize tls", err.Error())
	}

	// create helm client
	helmClient, err := helmutil.Initialize(s.TillerEndpoint, s.TillerKeyFile, s.TillerCertFile, s.TillerCAFile, clusterClient)
	if err != nil {
		klog.Errorf("failed to connect to tiller: %v", err)
	}

	clusterLabels, err := apilabels.ConvertSelectorToLabelsMap(strings.TrimSuffix(s.ClusterLabels, ","))
	if err != nil {
		klog.Errorf("Invalid label string: %s", s.ClusterLabels)
	}
	clusterLabels["name"] = s.ClusterName

	clusterAnnotations := map[string]string{
		prefixLabel: s.HelmReleasePrefix,
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(clusterClient, time.Minute*10)
	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		controllerClient,
		time.Minute*10,
		informers.WithNamespace(s.ClusterNamespace),
	)
	helmControl := helmutil.NewHelmControl(helmClient, controllerClient)

	// create and run resource mapper
	discoveryClient := cacheddiscovery.NewMemCacheClient(clusterClient.Discovery())
	mapper := restutils.NewMapper(discoveryClient, stopCh)
	mapper.Run()

	kubeControl := restutils.NewKubeControl(mapper, clusterCfg)

	// convert cluster name
	clusterName := strings.ToLower(strings.Replace(s.ClusterName, ".", "-", -1))
	var secretName = clusterName + "-federation-secret"
	clusterAnnotations["mcm.ibm.com/secretRef"] = secretName

	serverVersionString := serverVersion.String()
	if isOpenShift {
		if strings.Contains(serverVersionString, "+") {
			serverVersionString += ".rhos"
		} else {
			serverVersionString += "+rhos"
		}
	}
	if isAKS {
		if strings.Contains(serverVersionString, "+") {
			serverVersionString += ".aks"
		} else {
			serverVersionString += "+aks"
		}
	}

	routeV1Client, err := routev1.NewForConfig(clusterCfg)

	if err != nil {
		klog.Warningf("New config error: %v", err)
	}

	config := &klusterlet.Config{
		ClusterName:            clusterName,
		ClusterNamespace:       s.ClusterNamespace,
		MasterAddresses:        s.MasterAddresses,
		ServerVersion:          serverVersionString,
		Kubeconfig:             clusterCfg,
		KlusterletAddress:      s.KlusterletAddress,
		KlusterletIngress:      s.KlusterletIngress,
		KlusterletRoute:        s.KlusterletRoute,
		KlusterletPort:         s.KlusterletPort,
		KlusterletService:      s.KlusterletService,
		ClusterLabels:          clusterLabels,
		ClusterAnnotations:     clusterAnnotations,
		MonitoringScrapeTarget: s.DriverOptions.PrometheusScrapeTarget,
		EnableImpersonation:    s.EnableImpersonation,
	}

	klusterlet := klusterlet.NewKlusterlet(
		config,
		clusterClient,
		routeV1Client,
		controllerClient,
		kubeDynamicClientSet,
		controllerKubeClient,
		clusterClientSet,
		helmControl,
		kubeControl,
		kubeInformerFactory,
		informerFactory,
		stopCh)
	go klusterlet.Run(2)

	go kubeInformerFactory.Start(stopCh)
	go informerFactory.Start(stopCh)

	// start klusterlet server handler
	driverFactory := drivers.NewDriverFactory(s.DriverOptions, clusterClient)
	go klusterlet.ListenAndServe(driverFactory, net.ParseIP(s.Address), uint(s.Port), tlsOptions, nil, s.InSecure)

	return nil
}
