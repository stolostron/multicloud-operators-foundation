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

// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2018. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.

package helm

import (
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"

	"k8s.io/klog"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

// FakeHelmControl implements helm interface
type FakeHelmControl struct {
	helmclient helm.Interface
}

// NewFakeHelmControl creates a new helm control
func NewFakeHelmControl(helmclient helm.Interface) *FakeHelmControl {
	if helmclient == nil {
		return nil
	}

	return &FakeHelmControl{
		helmclient: helmclient,
	}
}

//CreateHelmRelease create a helm release
func (hc *FakeHelmControl) CreateHelmRelease(
	releaseName string,
	helmreponNamespace string,
	helmspec v1alpha1.HelmWorkSpec) (*rls.InstallReleaseResponse, error) {
	chart := &chart.Chart{}
	rls, err := hc.helmclient.InstallReleaseFromChart(
		chart,
		helmspec.Namespace,
		helm.ValueOverrides(helmspec.Values),
		helm.ReleaseName(releaseName),
		helm.InstallTimeout(300))
	if err != nil {
		return nil, err
	}
	return rls, nil
}

// UpdateHelmRelease updates a helmrelease
func (hc *FakeHelmControl) UpdateHelmRelease(
	releaseName string,
	helmreponNamespace string,
	helmspec v1alpha1.HelmWorkSpec) (*rls.UpdateReleaseResponse, error) {
	resp, err := hc.helmclient.UpdateRelease(
		releaseName,
		"",
		helm.UpdateValueOverrides(helmspec.Values),
		helm.UpgradeForce(true),
		helm.UpgradeTimeout(300),
	)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetHelmReleases get Helm Releases
func (hc *FakeHelmControl) GetHelmReleases(
	nameFilter string, codes []release.Status_Code, namespace string, limit int) (*rls.ListReleasesResponse, error) {
	var listRelOptions []helm.ReleaseListOption
	if nameFilter != "" {
		listRelOptions = append(listRelOptions, helm.ReleaseListFilter(nameFilter))
	}

	if len(codes) > 0 {
		listRelOptions = append(listRelOptions, helm.ReleaseListStatuses(codes))
	}

	if limit > 0 {
		listRelOptions = append(listRelOptions, helm.ReleaseListLimit(limit))
	}

	if namespace != "" {
		listRelOptions = append(listRelOptions, helm.ReleaseListNamespace(namespace))
	}

	rels, err := hc.helmclient.ListReleases(listRelOptions...)

	return rels, err
}

// DeleteHelmRelease delete helm release
func (hc *FakeHelmControl) DeleteHelmRelease(relName string) (*rls.UninstallReleaseResponse, error) {
	var urr *rls.UninstallReleaseResponse
	var err error
	if relName != "" {
		urr, err = hc.helmclient.DeleteRelease(
			relName,
			helm.DeletePurge(true),
			helm.DeleteTimeout(300),
		)
		if err != nil {
			klog.Errorf("Uninstall of helm release %s failed: %v", relName, err)
			return urr, err
		}
	}

	return urr, err
}
