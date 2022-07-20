package helpers

import (
	"testing"

	metainternal "k8s.io/apimachinery/pkg/apis/meta/internalversion"
)

func TestInternalListOptionsToSelectors(t *testing.T) {
	tests := []struct {
		name    string
		options *metainternal.ListOptions
	}{
		{"case1:", &metainternal.ListOptions{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InternalListOptionsToSelectors(tt.options)
		})
	}
}
