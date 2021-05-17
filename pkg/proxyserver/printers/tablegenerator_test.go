package printers

import (
	"fmt"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type TestPrintType struct {
	Data string
}

func (obj *TestPrintType) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (obj *TestPrintType) DeepCopyObject() runtime.Object {
	if obj == nil {
		return nil
	}
	clone := *obj
	return &clone
}

func PrintCustomType(obj *TestPrintType, options GenerateOptions) ([]metav1beta1.TableRow, error) {
	return []metav1beta1.TableRow{{Cells: []interface{}{obj.Data}}}, nil
}

func ErrorPrintHandler(obj *TestPrintType, options GenerateOptions) ([]metav1beta1.TableRow, error) {
	return nil, fmt.Errorf("ErrorPrintHandler error")
}

func TestCustomTypePrinting(t *testing.T) {
	columns := []metav1beta1.TableColumnDefinition{{Name: "Data"}}
	generator := NewTableGenerator()
	generator.TableHandler(columns, PrintCustomType)

	obj := TestPrintType{"test object"}
	table, err := generator.GenerateTable(&obj, GenerateOptions{})
	if err != nil {
		t.Fatalf("An error occurred generating the table for custom type: %#v", err)
	}

	expectedTable := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{{Name: "Data"}},
		Rows:              []metav1.TableRow{{Cells: []interface{}{"test object"}}},
	}
	if !reflect.DeepEqual(expectedTable, table) {
		t.Errorf("Error generating table from custom type. Expected (%#v), got (%#v)", expectedTable, table)
	}
}

func TestPrintHandlerError(t *testing.T) {
	columns := []metav1beta1.TableColumnDefinition{{Name: "Data"}}
	generator := NewTableGenerator()
	generator.TableHandler(columns, ErrorPrintHandler)
	obj := TestPrintType{"test object"}
	_, err := generator.GenerateTable(&obj, GenerateOptions{})
	if err == nil || err.Error() != "ErrorPrintHandler error" {
		t.Errorf("Did not get the expected error: %#v", err)
	}
}
