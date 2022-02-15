package addonfactory

import (
	"embed"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/agent"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

const AddonDefaultInstallNamespace = "open-cluster-management-agent-addon"

// AnnotationValuesName is the annotation Name of customized values
const AnnotationValuesName string = "addon.open-cluster-management.io/values"

type Values map[string]interface{}

type GetValuesFunc func(cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) (Values, error)

// AgentAddonFactory includes the common fields for building different agentAddon instances.
type AgentAddonFactory struct {
	scheme            *runtime.Scheme
	fs                embed.FS
	dir               string
	getValuesFuncs    []GetValuesFunc
	agentAddonOptions agent.AgentAddonOptions
}

// NewAgentAddonFactory builds an addonAgentFactory instance with addon name and fs.
// dir is the path prefix based on the fs path.
func NewAgentAddonFactory(addonName string, fs embed.FS, dir string) *AgentAddonFactory {
	return &AgentAddonFactory{
		fs:  fs,
		dir: dir,
		agentAddonOptions: agent.AgentAddonOptions{
			AddonName:       addonName,
			Registration:    nil,
			InstallStrategy: nil,
		},
	}
}

// WithScheme is an optional configuration, only used when the agentAddon has customized resource types.
func (f *AgentAddonFactory) WithScheme(scheme *runtime.Scheme) *AgentAddonFactory {
	f.scheme = scheme
	return f
}

// WithGetValuesFuncs adds a list of the getValues func.
// the values got from the big index Func will override the one from small index Func.
func (f *AgentAddonFactory) WithGetValuesFuncs(getValuesFuncs ...GetValuesFunc) *AgentAddonFactory {
	f.getValuesFuncs = getValuesFuncs
	return f
}

// WithInstallStrategy defines the installation strategy of the manifests prescribed by Manifests(..).
func (f *AgentAddonFactory) WithInstallStrategy(strategy *agent.InstallStrategy) *AgentAddonFactory {
	if strategy.InstallNamespace == "" {
		strategy.InstallNamespace = AddonDefaultInstallNamespace
	}
	f.agentAddonOptions.InstallStrategy = strategy

	return f
}

// WithAgentRegistrationOption defines how agent is registered to the hub cluster.
func (f *AgentAddonFactory) WithAgentRegistrationOption(option *agent.RegistrationOption) *AgentAddonFactory {
	f.agentAddonOptions.Registration = option
	return f
}

// BuildHelmAgentAddon builds a helm agentAddon instance.
func (f *AgentAddonFactory) BuildHelmAgentAddon() (agent.AgentAddon, error) {
	if f.scheme == nil {
		f.scheme = runtime.NewScheme()
	}
	_ = scheme.AddToScheme(f.scheme)
	_ = apiextensionsv1.AddToScheme(f.scheme)
	_ = apiextensionsv1beta1.AddToScheme(f.scheme)

	userChart, err := loadChart(f.fs, f.dir)
	if err != nil {
		return nil, err
	}
	// TODO: validate chart
	agentAddon := newHelmAgentAddon(f.scheme, userChart, f.getValuesFuncs, f.agentAddonOptions)

	return agentAddon, nil
}

// BuildTemplateAgentAddon builds a template agentAddon instance.
func (f *AgentAddonFactory) BuildTemplateAgentAddon() (agent.AgentAddon, error) {
	templateFiles, err := getTemplateFiles(f.fs, f.dir)
	if err != nil {
		klog.Errorf("failed to get template files. %v", err)
		return nil, err
	}
	if len(templateFiles) == 0 {
		return nil, fmt.Errorf("there is no template files")
	}

	if f.scheme == nil {
		f.scheme = runtime.NewScheme()
	}
	_ = scheme.AddToScheme(f.scheme)
	_ = apiextensionsv1.AddToScheme(f.scheme)
	_ = apiextensionsv1beta1.AddToScheme(f.scheme)

	agentAddon := newTemplateAgentAddon(f.scheme, f.getValuesFuncs, f.agentAddonOptions)

	for _, file := range templateFiles {
		template, err := f.fs.ReadFile(file)
		if err != nil {
			return nil, err
		}
		if err := agentAddon.validateTemplateData(file, template); err != nil {
			return nil, err
		}
		agentAddon.addTemplateData(file, template)
	}
	return agentAddon, nil
}
