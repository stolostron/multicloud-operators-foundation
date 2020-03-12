package v1beta1

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"k8s.io/apimachinery/pkg/conversion"
)

// Convert_mcm_WorkSpec_To_v1beta1_WorkSpec convert v1beta1 workspec to internal version
func Convert_mcm_WorkSpec_To_v1beta1_WorkSpec(out *mcm.WorkSpec, in *WorkSpec, s conversion.Scope) error {
	return autoConvert_mcm_WorkSpec_To_v1beta1_WorkSpec(out, in, s)
}

// Convert_v1beta1_WorkSpec_To_mcm_WorkSpec convert v1beta1 workspec to internal version
func Convert_v1beta1_WorkSpec_To_mcm_WorkSpec(in *WorkSpec, out *mcm.WorkSpec, s conversion.Scope) error {
	return nil
}

