// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// CloneAndAddLabel the given map and returns a new map with the given key and value added.
// Returns the given map, if labelKey is empty.
func CloneAndAddLabel(labels map[string]string, labelKey, labelValue string) map[string]string {
	if labelKey == "" {
		// Don't need to add a label.
		return labels
	}
	// Clone.
	newLabels := map[string]string{}
	for key, value := range labels {
		newLabels[key] = value
	}
	newLabels[labelKey] = labelValue
	return newLabels
}

// AddLabel returns a map with the given key and value added to the given map.
func AddLabel(labels map[string]string, labelKey, labelValue string) map[string]string {
	if labelKey == "" {
		// Don't need to add a label.
		return labels
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[labelKey] = labelValue
	return labels
}

//MatchLabelForLabelSelector match labels for labelselector, if labelSelecor is nil, select everything
func MatchLabelForLabelSelector(targetLabels map[string]string, labelSelector *metav1.LabelSelector) bool {
	selector, err := ConvertLabels(labelSelector)
	if err != nil {
		// this should not happen if the workset passed validation
		return false
	}
	if selector.Matches(labels.Set(targetLabels)) {
		return true
	}
	return false
}

func AddOwnersLabel(owners, resource, name, namespace string) string {
	if len(owners) == 0 {
		return resource + "." + namespace + "." + name
	}
	return owners + "," + resource + "." + namespace + "." + name
}

//string to map
func StringToMap(str string) map[string]string {
	if len(str) == 0 {
		return nil
	}
	returnMap := map[string]string{}
	split := strings.Split(str, ",")
	for _, sp := range split {
		tm := strings.Split(sp, "=")
		if len(tm) >= 2 {
			returnMap[tm[0]] = tm[1]
		}
	}
	return returnMap
}

func MergeMap(modified *bool, existing map[string]string, required map[string]string) {
	if existing == nil {
		existing = map[string]string{}
	}
	for k, v := range required {
		if existingV, ok := existing[k]; !ok || v != existingV {
			*modified = true
			existing[k] = v
		}
	}
}

// ConvertLabels returns label
func ConvertLabels(labelSelector *metav1.LabelSelector) (labels.Selector, error) {
	if labelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return labels.Nothing(), err
		}

		return selector, nil
	}

	return labels.Everything(), nil
}
