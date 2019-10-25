// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package authorization

import (
	"fmt"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
	"k8s.io/apiserver/pkg/authorization/path"
	"k8s.io/apiserver/pkg/authorization/union"
	"k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

// HcmDelegatingAuthorizationOptions is used to refactor DelegatingAuthorizationOptions
type HcmDelegatingAuthorizationOptions struct {
	DelegatingAuthorizationOptions *genericoptions.DelegatingAuthorizationOptions
	QPS                            float32
	Burst                          int
}

// ApplyTo is used to refactor ApplyTo of DelegatingAuthorizationOptions
func (hs HcmDelegatingAuthorizationOptions) ApplyTo(c *server.AuthorizationInfo) error {
	if hs.DelegatingAuthorizationOptions == nil {
		c.Authorizer = authorizerfactory.NewAlwaysAllowAuthorizer()
		return nil
	}

	client, err := hs.getClient()
	if err != nil {
		return err
	}

	c.Authorizer, err = hs.toAuthorizer(client)
	return err
}

// toAuthorizer is used to refactor toAuthorizer of DelegatingAuthorizationOptions
func (hs *HcmDelegatingAuthorizationOptions) toAuthorizer(client kubernetes.Interface) (authorizer.Authorizer, error) {
	var authorizers []authorizer.Authorizer

	if len(hs.DelegatingAuthorizationOptions.AlwaysAllowGroups) > 0 {
		authorizers = append(authorizers, authorizerfactory.NewPrivilegedGroups(hs.DelegatingAuthorizationOptions.AlwaysAllowGroups...))
	}

	if len(hs.DelegatingAuthorizationOptions.AlwaysAllowPaths) > 0 {
		a, err := path.NewAuthorizer(hs.DelegatingAuthorizationOptions.AlwaysAllowPaths)
		if err != nil {
			return nil, err
		}
		authorizers = append(authorizers, a)
	}

	if client == nil {
		klog.Warningf("No authorization-kubeconfig provided, so SubjectAccessReview of authorization tokens won't work.")
	} else {
		cfg := authorizerfactory.DelegatingAuthorizerConfig{
			SubjectAccessReviewClient: client.AuthorizationV1beta1().SubjectAccessReviews(),
			AllowCacheTTL:             hs.DelegatingAuthorizationOptions.AllowCacheTTL,
			DenyCacheTTL:              hs.DelegatingAuthorizationOptions.DenyCacheTTL,
		}
		delegatedAuthorizer, err := cfg.New()
		if err != nil {
			return nil, err
		}
		authorizers = append(authorizers, delegatedAuthorizer)
	}

	return union.New(authorizers...), nil
}

// getClient is used to refactor getClient of DelegatingAuthorizationOptions
func (hs *HcmDelegatingAuthorizationOptions) getClient() (kubernetes.Interface, error) {
	var clientConfig *rest.Config
	var err error
	if len(hs.DelegatingAuthorizationOptions.RemoteKubeConfigFile) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: hs.DelegatingAuthorizationOptions.RemoteKubeConfigFile}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

		clientConfig, err = loader.ClientConfig()
	} else {
		// without the remote kubeconfig file, try to use the in-cluster config.  Most addon API servers will
		// use this path. If it is optional, ignore errors.
		clientConfig, err = rest.InClusterConfig()
		if err != nil && hs.DelegatingAuthorizationOptions.RemoteKubeConfigFileOptional {
			if err != rest.ErrNotInCluster {
				klog.Warningf("failed to read in-cluster kubeconfig for delegated authorization: %v", err)
			}
			return nil, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get delegated authorization kubeconfig: %v", err)
	}

	// set high qps/burst limits since this will effectively limit API server responsiveness
	clientConfig.QPS = hs.QPS
	clientConfig.Burst = hs.Burst

	return kubernetes.NewForConfig(clientConfig)
}
