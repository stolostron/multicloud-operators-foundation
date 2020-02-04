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
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/proto/hapi/release"
)

// ConvertHelmReleaseFromRelease convert release to v1alpha1 releases
func ConvertHelmReleaseFromRelease(rls *release.Release) v1alpha1.HelmRelease {
	md := rls.GetChart().GetMetadata()
	firstDeployedSecs := rls.GetInfo().GetFirstDeployed().GetSeconds()
	lastDeployedSecs := rls.GetInfo().GetLastDeployed().GetSeconds()
	rl := v1alpha1.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name: rls.GetName(),
		},
		Spec: v1alpha1.HelmReleaseSpec{
			Namespace:     rls.GetNamespace(),
			Version:       rls.GetVersion(),
			Status:        rls.GetInfo().GetStatus().GetCode().String(),
			FirstDeployed: metav1.NewTime(time.Unix(firstDeployedSecs, 0)),
			LastDeployed:  metav1.NewTime(time.Unix(lastDeployedSecs, 0)),
			ChartName:     md.GetName(),
			ChartVersion:  md.GetVersion(),
			Description:   md.GetDescription(),
		},
	}

	return rl
}
