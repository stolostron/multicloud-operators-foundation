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

	hcmv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
)

// PrintReleaseTable print release table
func PrintReleaseTable(list *hcmv1alpha1.ResultHelmList) (*metav1beta1.Table, error) {
	rows, err := PrintReleaseList(list)
	if err != nil {
		return nil, err
	}

	columnDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Namespace", Type: "string"},
		{Name: "Status", Type: "string"},
		{Name: "Revision", Type: "string"},
		{Name: "Version", Type: "string"},
		{Name: "Chart", Type: "string"},
		{Name: "LastDeployed", Type: "string"},
		{Name: "FirstDeployed", Type: "string"},
	}

	return &metav1beta1.Table{
		Rows:              rows,
		ColumnDefinitions: columnDefinitions,
	}, nil
}

// PrintRelease print a single release
func PrintRelease(obj *hcmv1alpha1.HelmRelease) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}

	row.Cells = append(row.Cells, obj.Name, obj.Spec.Namespace, obj.Spec.Status, obj.Spec.Version, obj.Spec.ChartVersion, obj.Spec.ChartName,
		translateTimestamp(obj.Spec.LastDeployed), translateTimestamp(obj.Spec.FirstDeployed))
	return []metav1beta1.TableRow{row}, nil
}

// PrintReleaseList print a release list
func PrintReleaseList(list *hcmv1alpha1.ResultHelmList) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := PrintRelease(&list.Items[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

// translateTimestamp returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestamp(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}
	return duration.ShortHumanDuration(time.Since(timestamp.Time))
}
