package helpers

import (
	"time"

	apiconditions "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/conditions"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetStatusCondition sets the corresponding condition in conditions to newCondition.
func SetStatusCondition(conditions *[]apiconditions.Condition, newCondition apiconditions.Condition) {
	if conditions == nil {
		conditions = &[]apiconditions.Condition{}
	}
	existingCondition := FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		*conditions = append(*conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
	existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
}

// FindStatusCondition finds the conditionType in conditions.
func FindStatusCondition(conditions []apiconditions.Condition, conditionType apiconditions.ConditionType) *apiconditions.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}

	return nil
}

// RemoveStatusCondition removes the corresponding conditionType from conditions.
func RemoveStatusCondition(conditions *[]apiconditions.Condition, conditionType apiconditions.ConditionType) {
	if conditions == nil {
		return
	}
	var newConditions []apiconditions.Condition
	for _, condition := range *conditions {
		if condition.Type != conditionType {
			newConditions = append(newConditions, condition)
		}
	}

	*conditions = newConditions
}

// IsStatusConditionTrue returns true when the conditionType is present and set to `corev1.ConditionTrue`
func IsStatusConditionTrue(conditions []apiconditions.Condition, conditionType apiconditions.ConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionTrue)
}

// IsStatusConditionFalse returns true when the conditionType is present and set to `corev1.ConditionFalse`
func IsStatusConditionFalse(conditions []apiconditions.Condition, conditionType apiconditions.ConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionFalse)
}

// IsStatusConditionUnknown returns true when the conditionType is present and set to `corev1.ConditionUnknown`
func IsStatusConditionUnknown(conditions []apiconditions.Condition, conditionType apiconditions.ConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionUnknown)
}

// IsStatusConditionPresentAndEqual returns true when conditionType is present and equal to status.
func IsStatusConditionPresentAndEqual(conditions []apiconditions.Condition, conditionType apiconditions.ConditionType, status corev1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}

// SetClusterStatusCondition sets the corresponding condition in conditions to newCondition.
func SetClusterStatusCondition(conditions *[]clusterv1.StatusCondition, newCondition clusterv1.StatusCondition) {
	if conditions == nil {
		conditions = &[]clusterv1.StatusCondition{}
	}
	existingCondition := FindClusterStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		*conditions = append(*conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
	existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
}

// FindClusterStatusCondition finds the conditionType in conditions.
func FindClusterStatusCondition(conditions []clusterv1.StatusCondition, conditionType string) *clusterv1.StatusCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}

	return nil
}

// IsClusterStatusConditionTrue returns true when the conditionType is present and set to `metav1.ConditionTrue`
func IsClusterStatusConditionTrue(conditions []clusterv1.StatusCondition, conditionType string) bool {
	return IsClusterStatusConditionPresentAndEqual(conditions, conditionType, metav1.ConditionTrue)
}

// IsClusterStatusConditionFalse returns true when the conditionType is present and set to `metav1.ConditionFalse`
func IsClusterStatusConditionFalse(conditions []clusterv1.StatusCondition, conditionType string) bool {
	return IsClusterStatusConditionPresentAndEqual(conditions, conditionType, metav1.ConditionFalse)
}

// IsClusterStatusConditionPresentAndEqual returns true when conditionType is present and equal to status.
func IsClusterStatusConditionPresentAndEqual(conditions []clusterv1.StatusCondition, conditionType string, status metav1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}
