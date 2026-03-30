// Copyright Contributors to the Open Cluster Management project

package tlsprofile

import (
	"context"
	"crypto/tls"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// GetTLSConfig reads the TLS profile from the cluster's APIServer CR and returns a tls.Config.
// If not running on OpenShift (NotFound error), it returns a secure default (TLS 1.2).
// For other errors, it returns the error to allow the caller to decide how to handle it.
func GetTLSConfig(kubeConfig *rest.Config) (*tls.Config, error) {
	profile, err := GetTLSSecurityProfile(kubeConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Info("Not running on OpenShift cluster, using default TLS 1.2 configuration")
			return &tls.Config{MinVersion: tls.VersionTLS12}, nil
		}
		return nil, fmt.Errorf("failed to get TLS security profile: %w", err)
	}

	return ConvertTLSProfileToConfig(profile), nil
}

// GetTLSSecurityProfile reads the TLS security profile from the APIServer CR.
// Returns NotFound error if not running on OpenShift cluster.
func GetTLSSecurityProfile(kubeConfig *rest.Config) (*configv1.TLSSecurityProfile, error) {
	// Create an OpenShift config client
	configClient, err := configclient.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config client: %w", err)
	}

	// Get the APIServer CR
	apiServer, err := configClient.ConfigV1().APIServers().Get(context.TODO(), "cluster", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(4).Info("APIServer CR not found, not running on OpenShift cluster")
			return nil, err
		}
		return nil, fmt.Errorf("failed to get APIServer CR: %w", err)
	}

	// Return the TLS security profile
	if apiServer.Spec.TLSSecurityProfile == nil {
		klog.Info("No TLS security profile defined in APIServer CR, using Intermediate profile")
		// Default to Intermediate profile
		intermediate := configv1.TLSProfileIntermediateType
		return &configv1.TLSSecurityProfile{
			Type: intermediate,
		}, nil
	}

	return apiServer.Spec.TLSSecurityProfile, nil
}

// ConvertTLSProfileToConfig converts an OpenShift TLSSecurityProfile to a Go tls.Config.
// If profile is nil, returns a secure default configuration with TLS 1.2.
func ConvertTLSProfileToConfig(profile *configv1.TLSSecurityProfile) *tls.Config {
	if profile == nil {
		return &tls.Config{MinVersion: tls.VersionTLS12}
	}

	var minVersion uint16
	var cipherSuites []uint16

	switch profile.Type {
	case configv1.TLSProfileOldType:
		// #nosec G402 -- TLS 1.0 is set based on OpenShift cluster admin configuration
		// This respects the cluster-wide TLS profile set in the APIServer CR
		minVersion = tls.VersionTLS10
		cipherSuites = getCipherSuites(configv1.TLSProfiles[configv1.TLSProfileOldType].Ciphers)
	case configv1.TLSProfileIntermediateType:
		minVersion = tls.VersionTLS12
		cipherSuites = getCipherSuites(configv1.TLSProfiles[configv1.TLSProfileIntermediateType].Ciphers)
	case configv1.TLSProfileModernType:
		minVersion = tls.VersionTLS13
		// TLS 1.3 doesn't use configurable cipher suites
		cipherSuites = nil
	case configv1.TLSProfileCustomType:
		if profile.Custom != nil {
			// #nosec G402 -- MinTLSVersion is set based on OpenShift cluster admin configuration
			minVersion = getTLSVersion(string(profile.Custom.MinTLSVersion))
			cipherSuites = getCipherSuites(profile.Custom.Ciphers)
		} else {
			klog.Warning("Custom TLS profile specified but no custom configuration found, using TLS 1.2")
			minVersion = tls.VersionTLS12
		}
	default:
		klog.Warningf("Unknown TLS profile type: %s, using TLS 1.2", profile.Type)
		minVersion = tls.VersionTLS12
	}

	// #nosec G402 -- MinVersion is set based on OpenShift cluster TLS profile configuration
	config := &tls.Config{
		MinVersion: minVersion,
	}

	if len(cipherSuites) > 0 {
		config.CipherSuites = cipherSuites
	}

	klog.Infof("Applied TLS profile: type=%s, minVersion=%s, cipherCount=%d",
		profile.Type, getTLSVersionName(minVersion), len(cipherSuites))

	return config
}

// getTLSVersion converts OpenShift TLS version string to Go constant.
// #nosec G402 -- This function returns TLS versions based on OpenShift cluster configuration
func getTLSVersion(version string) uint16 {
	switch configv1.TLSProtocolVersion(version) {
	case configv1.VersionTLS10:
		return tls.VersionTLS10
	case configv1.VersionTLS11:
		return tls.VersionTLS11
	case configv1.VersionTLS12:
		return tls.VersionTLS12
	case configv1.VersionTLS13:
		return tls.VersionTLS13
	default:
		klog.Warningf("Unknown TLS version: %s, defaulting to TLS 1.2", version)
		return tls.VersionTLS12
	}
}

// getTLSVersionName returns a human-readable name for a TLS version.
func getTLSVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// getCipherSuites converts OpenShift cipher suite names to Go constants.
func getCipherSuites(cipherNames []string) []uint16 {
	var ciphers []uint16
	cipherMap := getCipherSuiteMap()

	for _, name := range cipherNames {
		if cipher, ok := cipherMap[name]; ok {
			ciphers = append(ciphers, cipher)
		} else {
			klog.Warningf("Unknown cipher suite: %s", name)
		}
	}

	return ciphers
}

// getCipherSuiteMap returns a mapping of OpenShift cipher names to Go constants.
// Supports both OpenShift format (ECDHE-RSA-AES128-GCM-SHA256) and Go format (TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256).
func getCipherSuiteMap() map[string]uint16 {
	return map[string]uint16{
		// Go format (with underscores and TLS_ prefix)
		"TLS_RSA_WITH_AES_128_CBC_SHA":                  tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":                  tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":               tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":               tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_AES_128_GCM_SHA256":                        tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":                        tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256":                  tls.TLS_CHACHA20_POLY1305_SHA256,

		// OpenShift format (with hyphens, no TLS_ prefix)
		"ECDHE-ECDSA-AES128-GCM-SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"ECDHE-RSA-AES128-GCM-SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"ECDHE-ECDSA-AES256-GCM-SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"ECDHE-RSA-AES256-GCM-SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"ECDHE-ECDSA-CHACHA20-POLY1305": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		"ECDHE-RSA-CHACHA20-POLY1305":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"ECDHE-ECDSA-AES128-SHA256":     tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"ECDHE-RSA-AES128-SHA256":       tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"ECDHE-ECDSA-AES128-SHA":        tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"ECDHE-RSA-AES128-SHA":          tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"ECDHE-ECDSA-AES256-SHA384":     tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"ECDHE-RSA-AES256-SHA384":       tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"ECDHE-ECDSA-AES256-SHA":        tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"ECDHE-RSA-AES256-SHA":          tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"AES128-GCM-SHA256":             tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"AES256-GCM-SHA384":             tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"AES128-SHA256":                 tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"AES256-SHA256":                 tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"AES128-SHA":                    tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"AES256-SHA":                    tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"DES-CBC3-SHA":                  tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	}
}
