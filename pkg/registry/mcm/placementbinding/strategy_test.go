// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
package placementbinding

import (
	"testing"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var goodpb = &mcm.PlacementBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pb",
		Namespace: "default",
	},
	PlacementPolicyRef: mcm.PlacementPolicyRef{
		Name:     "pp",
		Kind:     "PlacementPolicy",
		APIGroup: "mcm.ibm.com",
	},
	Subjects: []mcm.Subject{
		mcm.Subject{
			Name:     "sub",
			Kind:     "Deployable",
			APIGroup: "mcm.ibm.com",
		},
	},
}

func TestValidatePlacementBinding(t *testing.T) {

	err := validatePlacementBinding(goodpb)
	if err != nil {
		t.Errorf("should not fail to validate good placementbinding")
	}

	badpb := goodpb.DeepCopy()
	badpb.PlacementPolicyRef.Name = ""
	err = validatePlacementBinding(badpb)
	if err == nil {
		t.Errorf("should failed to validate placementbinding - empty ref name")
	}

	badpb = goodpb.DeepCopy()
	badpb.PlacementPolicyRef.Kind = ""
	err = validatePlacementBinding(badpb)
	if err == nil {
		t.Errorf("should failed to validate placementbinding - empty ref kind")
	}

	badpb = goodpb.DeepCopy()
	badpb.PlacementPolicyRef.APIGroup = ""
	err = validatePlacementBinding(badpb)
	if err == nil {
		t.Errorf("should failed to validate placementbinding - empty ref apigroup")
	}

	badpb = goodpb.DeepCopy()
	badpb.Subjects[0].Name = ""
	err = validatePlacementBinding(badpb)
	if err == nil {
		t.Errorf("should failed to validate placementbinding - empty subject name")
	}

	badpb = goodpb.DeepCopy()
	badpb.Subjects[0].Kind = ""
	err = validatePlacementBinding(badpb)
	if err == nil {
		t.Errorf("should failed to validate placementbinding - empty subject kind")
	}

	badpb = goodpb.DeepCopy()
	badpb.Subjects[0].APIGroup = ""
	err = validatePlacementBinding(badpb)
	if err == nil {
		t.Errorf("should failed to validate placementbinding - empty subject apigroup")
	}
}
