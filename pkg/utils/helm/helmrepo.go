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

package helm

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
	"k8s.io/klog"
)

// DefaultHelmHome is the default HELM_HOME.
var DefaultHelmHome = helmpath.Home(filepath.Join("/tmp/hcm", ".helm"))

// GenerateSettings generate helm settings
func GenerateSettings() helm_env.EnvSettings {
	settings := helm_env.EnvSettings{
		Home: DefaultHelmHome,
	}

	return settings
}

// EnsureDefaultRepos ensures default repo exists
func EnsureDefaultRepos(home helmpath.Home) error {
	repoFile := home.RepositoryFile()
	if fi, err := os.Stat(repoFile); err != nil {
		klog.Infof("Creating %s \n", repoFile)
		f := repo.NewRepoFile()
		fileModePerm := os.FileMode(0644)
		if err := f.WriteFile(repoFile, fileModePerm); err != nil {
			return err
		}
	} else if fi.IsDir() {
		return fmt.Errorf("%s must be a file, not a directory", repoFile)
	}
	return nil
}

// EnsureDirectories checks to see if $HELM_HOME exists.
//
// If $HELM_HOME does not exist, this function will create it.
func EnsureDirectories(home helmpath.Home) error {
	configDirectories := []string{
		home.String(),
		home.Repository(),
		home.Cache(),
		home.LocalRepository(),
		home.Plugins(),
		home.Starters(),
		home.Archive(),
	}
	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			if err := os.MkdirAll(p, 0755); err != nil {
				return fmt.Errorf("could not create %s: %s", p, err)
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("%s must be a directory", p)
		}
	}

	return nil
}

// LocateChartPath looks for a chart directory in known places, and returns either the full path or an error.
//
// This does not ensure that the chart is well-formed; only that the requested filename exists.
//
// Order of resolution:
// - current working directory
// - if path is absolute or begins with '.', error out here
// - chart repos in $HELM_HOME
// - URL
//
// If 'verify' is true, this will attempt to also verify the chart.
func LocateChartPath(repoURL, username, password, name, version string, verify, insecure bool, keyring,
	certFile, keyFile, caFile string) (string, error) {
	settings := GenerateSettings()
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if fi, err := os.Stat(name); err == nil {
		abs, err := filepath.Abs(name)
		if err != nil {
			return abs, err
		}
		if verify {
			if fi.IsDir() {
				return "", errors.New("cannot verify a directory")
			}
			if _, err := downloader.VerifyChart(abs, keyring); err != nil {
				return "", err
			}
		}
		return abs, nil
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, ".") {
		return name, fmt.Errorf("path %q not found", name)
	}

	crepo := filepath.Join(settings.Home.Repository(), name)
	if _, err := os.Stat(crepo); err == nil {
		return filepath.Abs(crepo)
	}

	if repoURL != "" {
		chartURL, err := repo.FindChartInAuthRepoURL(repoURL, username, password, name, version,
			certFile, keyFile, caFile, getProviders(insecure, settings))
		if err != nil {
			return "", err
		}
		name = chartURL
	}

	// if it is InSecure, the helm repo is on the hub, so replace host in chartURL by host in repoURL
	if insecure {
		rurl, err := url.Parse(repoURL)
		if err != nil {
			return "", err
		}
		curl, err := url.Parse(name)
		if err != nil {
			return "", err
		}
		curl.Host = rurl.Host
		name = curl.String()
	}

	return DownloadChart(name, version, username, password, keyring, verify, insecure)
}

// DownloadChart download chart from a give url
func DownloadChart(chartURL, version, username, password, keyring string, verify, insecure bool) (string, error) {
	settings := GenerateSettings()
	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Out:      os.Stdout,
		Keyring:  keyring,
		Getters:  getProviders(insecure, settings),
		Username: username,
		Password: password,
	}
	if verify {
		dl.Verify = downloader.VerifyAlways
	}

	if _, err := os.Stat(settings.Home.Archive()); os.IsNotExist(err) {
		if err := os.MkdirAll(settings.Home.Archive(), 0744); err != nil {
			return "", err
		}
	}

	filename, _, err := dl.DownloadTo(chartURL, version, settings.Home.Archive())
	if err == nil {
		lname, fileerr := filepath.Abs(filename)
		if fileerr != nil {
			return filename, fileerr
		}
		return lname, nil
	} else if settings.Debug {
		return filename, err
	}
	return filename, fmt.Errorf("failed to download %q (hint: running `helm repo update` may help)", chartURL)
}

func getProviders(insecure bool, settings helm_env.EnvSettings) getter.Providers {
	if insecure {
		return NewInSecureProviders()
	}
	return getter.All(settings)
}
