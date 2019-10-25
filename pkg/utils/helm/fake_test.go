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
	"testing"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"k8s.io/helm/pkg/helm"
	helmrelease "k8s.io/helm/pkg/proto/hapi/release"
)

func TestFakeHelmControl(t *testing.T) {
	helmclient := &helm.FakeClient{}
	helmcontrol := NewFakeHelmControl(helmclient)
	helmspec := v1alpha1.HelmWorkSpec{
		ReleaseName: "r1",
		ChartURL:    "https://raw.githubusercontent.com/abdasgupta/helm-repo/master/3.1-mcm-guestbook/gbf-0.1.0.tgz",
		Namespace:   "kube-system",
	}

	//Test createhelmrelease
	rls, err := helmcontrol.CreateHelmRelease("r1", "rn1", helmspec)
	if err != nil {
		t.Errorf("error: %+v", err)
	}
	if rls.GetRelease() == nil {
		t.Errorf("Can not get the release")
	}

	statusCode := []helmrelease.Status_Code{
		helmrelease.Status_UNKNOWN,
		helmrelease.Status_DEPLOYED,
		helmrelease.Status_DELETED,
		helmrelease.Status_DELETING,
		helmrelease.Status_FAILED,
		helmrelease.Status_PENDING_INSTALL,
		helmrelease.Status_PENDING_UPGRADE,
		helmrelease.Status_PENDING_ROLLBACK,
	}
	//test get helm releases
	releases, e := helmcontrol.GetHelmReleases("", statusCode, "", 256)
	if e != nil {
		t.Errorf("error: %+v", e)
	}
	if releases.Count != 1 {
		t.Errorf("releases.Count: %+v", releases.Count)
	}

	_, e = helmcontrol.DeleteHelmRelease("r1")
	if e != nil {
		t.Errorf("error: %+v", e)
	}
	releases, e = helmcontrol.GetHelmReleases("", statusCode, "", 256)
	if e != nil {
		t.Errorf("error: %+v", e)
	}
	if releases.Count != 0 {
		t.Errorf("releases.Count: %+v", releases.Count)
	}
}
