package utils

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// MatchLabelForLabelSelector match labels for labelselector, if labelSelecor is nil, select everything
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

func MergeMap(modified *bool, existing *map[string]string, required map[string]string) {
	if *existing == nil {
		*existing = map[string]string{}
	}
	for k, v := range required {
		actualKey := k
		removeKey := false

		// if "required" map contains a key with "-" as suffix, remove that
		// key from the existing map instead of replacing the value
		if strings.HasSuffix(k, "-") {
			removeKey = true
			actualKey = strings.TrimRight(k, "-")
		}

		if existingV, ok := (*existing)[actualKey]; removeKey {
			if !ok {
				continue
			}
			// value found -> it should be removed
			delete(*existing, actualKey)
			*modified = true

		} else if !ok || v != existingV {
			*modified = true
			(*existing)[actualKey] = v
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

// ContainsString to check string from a slice of strings.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// ContainsString to remove string from a slice of strings.
func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// SyncMapField sync the "syncFiledKey" label filed of required map.
func SyncMapField(modified *bool, existing *map[string]string, required map[string]string, syncFiledKey string) {
	*modified = false
	if *existing == nil {
		*existing = map[string]string{}
	}

	// value found in existing
	if _, ok := (*existing)[syncFiledKey]; ok {
		// value not found in required -> it should be removed from existing
		if required == nil || len(required[syncFiledKey]) == 0 {
			delete(*existing, syncFiledKey)
			*modified = true
			return
		}

		// value found in required
		if (*existing)[syncFiledKey] != required[syncFiledKey] {
			(*existing)[syncFiledKey] = required[syncFiledKey]
			*modified = true
			return
		}
		return
	}

	// value do not in existing, but in required
	if required != nil && len(required[syncFiledKey]) != 0 {
		(*existing)[syncFiledKey] = required[syncFiledKey]
		*modified = true
		return
	}

}
