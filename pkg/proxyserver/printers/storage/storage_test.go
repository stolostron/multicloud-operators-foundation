package storage

import (
	"context"
	"testing"

	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/printers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/printers/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestConvertToToable(t *testing.T) {
	tableOptions := metav1.TableOptions{}
	tests := []struct {
		name         string
		tableOptions runtime.Object
	}{
		{"case1:", tableOptions.DeepCopyObject()},
		{"case2:", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := TableConvertor{TableGenerator: printers.NewTableGenerator().With(internalversion.AddHandlers)}
			tc.ConvertToTable(context.TODO(), nil, tt.tableOptions)
		})
	}
}
