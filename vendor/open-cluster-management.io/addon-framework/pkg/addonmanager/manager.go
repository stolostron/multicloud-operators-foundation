package addonmanager

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/openshift/library-go/pkg/operator/events"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/constants"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/controllers/addonhealthcheck"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/controllers/addoninstall"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/controllers/agentdeploy"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/controllers/certificate"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/controllers/clustermanagement"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/controllers/registration"
	"open-cluster-management.io/addon-framework/pkg/agent"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	addoninformers "open-cluster-management.io/api/client/addon/informers/externalversions"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	workv1client "open-cluster-management.io/api/client/work/clientset/versioned"
	workv1informers "open-cluster-management.io/api/client/work/informers/externalversions"
)

// AddonManager is the interface to initialize a manager on hub to manage the addon
// agents on all managedcluster
type AddonManager interface {
	// AddAgent register an addon agent to the manager.
	AddAgent(addon agent.AgentAddon) error

	// Start starts all registered addon agent.
	Start(ctx context.Context) error
}

type addonManager struct {
	addonAgents map[string]agent.AgentAddon
	config      *rest.Config
}

func (a *addonManager) AddAgent(addon agent.AgentAddon) error {
	addonOption := addon.GetAgentAddonOptions()
	if len(addonOption.AddonName) == 0 {
		return fmt.Errorf("Addon name should be set")
	}
	if _, ok := a.addonAgents[addonOption.AddonName]; ok {
		return fmt.Errorf("An agent is added for the addon already")
	}
	a.addonAgents[addonOption.AddonName] = addon
	return nil
}

func (a *addonManager) Start(ctx context.Context) error {
	kubeClient, err := kubernetes.NewForConfig(a.config)
	if err != nil {
		return err
	}

	addonClient, err := addonv1alpha1client.NewForConfig(a.config)
	if err != nil {
		return err
	}

	clusterClient, err := clusterv1client.NewForConfig(a.config)
	if err != nil {
		return err
	}

	workClient, err := workv1client.NewForConfig(a.config)
	if err != nil {
		return err
	}

	namespace, err := a.getComponentNamespace()
	if err != nil {
		klog.Warningf("unable to identify the current namespace for events: %v", err)
	}
	controllerRef, err := events.GetControllerReferenceForCurrentPod(ctx, kubeClient, namespace, nil)
	if err != nil {
		klog.Warningf("unable to get owner reference (falling back to namespace): %v", err)
	}

	eventRecorder := events.NewKubeRecorder(
		kubeClient.CoreV1().Events(namespace), "addon", controllerRef)

	addonNames := []string{}
	for key := range a.addonAgents {
		addonNames = append(addonNames, key)
	}
	addonInformers := addoninformers.NewSharedInformerFactory(addonClient, 10*time.Minute)
	workInformers := workv1informers.NewSharedInformerFactoryWithOptions(workClient, 10*time.Minute,
		workv1informers.WithTweakListOptions(func(listOptions *metav1.ListOptions) {
			selector := &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      constants.AddonLabel,
						Operator: metav1.LabelSelectorOpIn,
						Values:   addonNames,
					},
				},
			}
			listOptions.LabelSelector = metav1.FormatLabelSelector(selector)
		}),
	)
	clusterInformers := clusterv1informers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
	kubeInfomers := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClient, 10*time.Minute,
		kubeinformers.WithTweakListOptions(func(listOptions *metav1.ListOptions) {
			selector := &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      constants.AddonLabel,
						Operator: metav1.LabelSelectorOpIn,
						Values:   addonNames,
					},
				},
			}
			listOptions.LabelSelector = metav1.FormatLabelSelector(selector)
		}),
	)

	deployController := agentdeploy.NewAddonDeployController(
		workClient,
		addonClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		addonInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		workInformers.Work().V1().ManifestWorks(),
		a.addonAgents,
		eventRecorder,
	)
	hookDeployController := agentdeploy.NewAddonHookDeployController(
		workClient,
		addonClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		addonInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		workInformers.Work().V1().ManifestWorks(),
		a.addonAgents,
		eventRecorder,
	)

	registrationController := registration.NewAddonConfigurationController(
		addonClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		addonInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		a.addonAgents,
		eventRecorder,
	)

	csrApproveController := certificate.NewCSRApprovingController(
		kubeClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		kubeInfomers.Certificates().V1().CertificateSigningRequests(),
		addonInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		a.addonAgents,
		eventRecorder,
	)

	csrSignController := certificate.NewCSRSignController(
		kubeClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		kubeInfomers.Certificates().V1().CertificateSigningRequests(),
		addonInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		a.addonAgents,
		eventRecorder,
	)

	clusterManagementController := clustermanagement.NewClusterManagementController(
		addonClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		addonInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		addonInformers.Addon().V1alpha1().ClusterManagementAddOns(),
		a.addonAgents,
		eventRecorder,
	)

	addonInstallController := addoninstall.NewAddonInstallController(
		addonClient,
		clusterInformers.Cluster().V1().ManagedClusters(),
		addonInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		a.addonAgents,
		eventRecorder,
	)

	addonHealthCheckController := addonhealthcheck.NewAddonHealthCheckController(
		addonClient,
		addonInformers.Addon().V1alpha1().ManagedClusterAddOns(),
		a.addonAgents,
		eventRecorder)

	go addonInformers.Start(ctx.Done())
	go workInformers.Start(ctx.Done())
	go clusterInformers.Start(ctx.Done())
	go kubeInfomers.Start(ctx.Done())

	go deployController.Run(ctx, 1)
	go hookDeployController.Run(ctx, 1)
	go registrationController.Run(ctx, 1)
	go csrApproveController.Run(ctx, 1)
	go csrSignController.Run(ctx, 1)
	go clusterManagementController.Run(ctx, 1)
	go addonInstallController.Run(ctx, 1)
	go addonHealthCheckController.Run(ctx, 1)
	return nil
}

func (a *addonManager) getComponentNamespace() (string, error) {
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "open-cluster-management", err
	}
	return string(nsBytes), nil
}

// New returns a new Manager for creating addon agents.
func New(config *rest.Config) (AddonManager, error) {
	return &addonManager{
		config:      config,
		addonAgents: map[string]agent.AgentAddon{},
	}, nil
}
