package helpers

import (
	"testing"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	apiconditions "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/conditions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConditions(t *testing.T) {
	var conditions []apiconditions.Condition

	newCondition := apiconditions.Condition{
		Type:    "condition1",
		Status:  corev1.ConditionFalse,
		Reason:  "reason1",
		Message: "failed conditions",
	}

	SetStatusCondition(&conditions, newCondition)

	existCondition := FindStatusCondition(conditions, newCondition.Type)
	if existCondition == nil {
		t.Errorf("failed to find condition")
	}
	newCondition.Status = corev1.ConditionTrue
	SetStatusCondition(&conditions, newCondition)
	if existCondition == nil {
		t.Errorf("failed to find condition")
	}

	if !IsStatusConditionTrue(conditions, existCondition.Type) {
		t.Errorf("condistion status is true")
	}

	if IsStatusConditionFalse(conditions, existCondition.Type) {
		t.Errorf("condistion status is true")
	}

	if IsStatusConditionUnknown(conditions, existCondition.Type) {
		t.Errorf("condistion status is true")
	}

	if IsStatusConditionPresentAndEqual(conditions, existCondition.Type, corev1.ConditionFalse) {
		t.Errorf("condistion status is true")
	}

	RemoveStatusCondition(&conditions, existCondition.Type)
	existCondition = FindStatusCondition(conditions, newCondition.Type)
	if existCondition != nil {
		t.Errorf("failed to remove condition")
	}
}

func TestClusterStatusConditions(t *testing.T) {
	var conditions []clusterv1.StatusCondition

	newCondition := clusterv1.StatusCondition{
		Type:    "condition1",
		Status:  metav1.ConditionFalse,
		Reason:  "reason1",
		Message: "failed conditions",
	}

	SetClusterStatusCondition(&conditions, newCondition)
	if existCondition := FindClusterStatusCondition(conditions, newCondition.Type); existCondition == nil {
		t.Errorf("failed to find condition")
	}
	newCondition.Status = metav1.ConditionTrue
	SetClusterStatusCondition(&conditions, newCondition)
	existCondition := FindClusterStatusCondition(conditions, newCondition.Type)
	if existCondition == nil {
		t.Errorf("failed to find condition")
	}

	if !IsClusterStatusConditionTrue(conditions, existCondition.Type) {
		t.Errorf("condistion status is true")
	}

	if IsClusterStatusConditionFalse(conditions, existCondition.Type) {
		t.Errorf("condistion status is true")
	}
}
