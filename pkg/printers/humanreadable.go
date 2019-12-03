// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package printers

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type TablePrinter interface {
	PrintTable(obj runtime.Object, options PrintOptions) (*metav1beta1.Table, error)
}

type PrintHandler interface {
	Handler(columns, columnsWithWide []string, printFunc interface{}) error
	TableHandler(columns []metav1beta1.TableColumnDefinition, printFunc interface{}) error
	DefaultTableHandler(columns []metav1beta1.TableColumnDefinition, printFunc interface{}) error
}

type handlerEntry struct {
	columnDefinitions []metav1beta1.TableColumnDefinition
	printRows         bool
	printFunc         reflect.Value
}

// HumanReadablePrinter is an implementation of ResourcePrinter which attempts to provide
// more elegant output. It is not threadsafe, but you may call PrintObj repeatedly; headers
// will only be printed if the object type changes. This makes it useful for printing items
// received from watches.
type HumanReadablePrinter struct {
	handlerMap     map[reflect.Type]*handlerEntry
	defaultHandler *handlerEntry
	options        PrintOptions
	lastType       interface{}
	skipTabWriter  bool
	decoder        runtime.Decoder
}

var _ PrintHandler = &HumanReadablePrinter{}

// NewHumanReadablePrinter creates a HumanReadablePrinter.
// If encoder and decoder are provided, an attempt to convert unstructured types to internal types is made.
func NewHumanReadablePrinter(decoder runtime.Decoder, options PrintOptions) *HumanReadablePrinter {
	printer := &HumanReadablePrinter{
		handlerMap: make(map[reflect.Type]*handlerEntry),
		options:    options,
		decoder:    decoder,
	}
	return printer
}

// NewTablePrinter creates a HumanReadablePrinter suitable for calling PrintTable().
func NewTablePrinter() *HumanReadablePrinter {
	return &HumanReadablePrinter{
		handlerMap: make(map[reflect.Type]*handlerEntry),
	}
}

// AddTabWriter sets whether the PrintObj function will format with tabwriter (true
// by default).
func (h *HumanReadablePrinter) AddTabWriter(t bool) *HumanReadablePrinter {
	h.skipTabWriter = !t
	return h
}

func (h *HumanReadablePrinter) With(fns ...func(PrintHandler)) *HumanReadablePrinter {
	for _, fn := range fns {
		fn(h)
	}
	return h
}

// EnsurePrintHeaders sets the HumanReadablePrinter option "NoHeaders" to false
// and removes the .lastType that was printed, which forces headers to be
// printed in cases where multiple lists of the same resource are printed
// consecutively, but are separated by non-printer related information.
func (h *HumanReadablePrinter) EnsurePrintHeaders() {
	h.options.NoHeaders = false
	h.lastType = nil
}

// Handler adds a print handler with a given set of columns to HumanReadablePrinter instance.
// See ValidatePrintHandlerFunc for required method signature.
func (h *HumanReadablePrinter) Handler(columns, columnsWithWide []string, printFunc interface{}) error {
	var columnDefinitions []metav1beta1.TableColumnDefinition
	for i, column := range columns {
		format := ""
		if i == 0 && strings.EqualFold(column, "name") {
			format = "name"
		}

		columnDefinitions = append(columnDefinitions, metav1beta1.TableColumnDefinition{
			Name:        column,
			Description: column,
			Type:        "string",
			Format:      format,
		})
	}
	for _, column := range columnsWithWide {
		columnDefinitions = append(columnDefinitions, metav1beta1.TableColumnDefinition{
			Name:        column,
			Description: column,
			Type:        "string",
			Priority:    1,
		})
	}

	printFuncValue := reflect.ValueOf(printFunc)
	if err := ValidatePrintHandlerFunc(printFuncValue); err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to register print function: %v", err))
		return err
	}

	entry := &handlerEntry{
		columnDefinitions: columnDefinitions,
		printFunc:         printFuncValue,
	}

	objType := printFuncValue.Type().In(0)
	if _, ok := h.handlerMap[objType]; ok {
		err := fmt.Errorf("registered duplicate printer for %v", objType)
		utilruntime.HandleError(err)
		return err
	}
	h.handlerMap[objType] = entry
	return nil
}

// TableHandler adds a print handler with a given set of columns to HumanReadablePrinter instance.
// See ValidateRowPrintHandlerFunc for required method signature.
func (h *HumanReadablePrinter) TableHandler(columnDefinitions []metav1beta1.TableColumnDefinition, printFunc interface{}) error {
	printFuncValue := reflect.ValueOf(printFunc)
	if err := ValidateRowPrintHandlerFunc(printFuncValue); err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to register print function: %v", err))
		return err
	}
	entry := &handlerEntry{
		columnDefinitions: columnDefinitions,
		printRows:         true,
		printFunc:         printFuncValue,
	}

	objType := printFuncValue.Type().In(0)
	if _, ok := h.handlerMap[objType]; ok {
		err := fmt.Errorf("registered duplicate printer for %v", objType)
		utilruntime.HandleError(err)
		return err
	}
	h.handlerMap[objType] = entry
	return nil
}

// DefaultTableHandler registers a set of columns and a print func that is given a chance to process
// any object without an explicit handler. Only the most recently set print handler is used.
// See ValidateRowPrintHandlerFunc for required method signature.
func (h *HumanReadablePrinter) DefaultTableHandler(
	columnDefinitions []metav1beta1.TableColumnDefinition, printFunc interface{}) error {
	printFuncValue := reflect.ValueOf(printFunc)
	if err := ValidateRowPrintHandlerFunc(printFuncValue); err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to register print function: %v", err))
		return err
	}
	entry := &handlerEntry{
		columnDefinitions: columnDefinitions,
		printRows:         true,
		printFunc:         printFuncValue,
	}

	h.defaultHandler = entry
	return nil
}

// ValidateRowPrintHandlerFunc validates print handler signature.
// printFunc is the function that will be called to print an object.
// It must be of the following type:
//  func printFunc(object ObjectType, options PrintOptions) ([]metav1beta1.TableRow, error)
// where ObjectType is the type of the object that will be printed, and the first
// return value is an array of rows, with each row containing a number of cells that
// match the number of columns defined for that printer function.
func ValidateRowPrintHandlerFunc(printFunc reflect.Value) error {
	if printFunc.Kind() != reflect.Func {
		return fmt.Errorf("invalid print handler. %#v is not a function", printFunc)
	}
	funcType := printFunc.Type()
	if funcType.NumIn() != 2 || funcType.NumOut() != 2 {
		return fmt.Errorf("invalid print handler. Must accept 2 parameters and return 2 value")
	}
	if funcType.In(1) != reflect.TypeOf((*PrintOptions)(nil)).Elem() ||
		funcType.Out(0) != reflect.TypeOf((*[]metav1beta1.TableRow)(nil)).Elem() ||
		funcType.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf("invalid print handler. The expected signature is: "+
			"func handler(obj %v, options PrintOptions) ([]metav1beta1.TableRow, error)", funcType.In(0))
	}
	return nil
}

// ValidatePrintHandlerFunc validates print handler signature.
// printFunc is the function that will be called to print an object.
// It must be of the following type:
//  func printFunc(object ObjectType, w io.Writer, options PrintOptions) error
// where ObjectType is the type of the object that will be printed.
// Deprecated: will be replaced with ValidateRowPrintHandlerFunc
func ValidatePrintHandlerFunc(printFunc reflect.Value) error {
	if printFunc.Kind() != reflect.Func {
		return fmt.Errorf("invalid print handler. %#v is not a function", printFunc)
	}
	funcType := printFunc.Type()
	if funcType.NumIn() != 3 || funcType.NumOut() != 1 {
		return fmt.Errorf("invalid print handler. Must accept 3 parameters and return 1 value")
	}
	if funcType.In(1) != reflect.TypeOf((*io.Writer)(nil)).Elem() ||
		funcType.In(2) != reflect.TypeOf((*PrintOptions)(nil)).Elem() ||
		funcType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf("invalid print handler. The expected signature is: "+
			"func handler(obj %v, w io.Writer, options PrintOptions) error", funcType.In(0))
	}
	return nil
}

func (h *HumanReadablePrinter) HandledResources() []string {
	keys := make([]string, 0)

	for k := range h.handlerMap {
		// k.String looks like "*api.PodList" and we want just "pod"
		api := strings.Split(k.String(), ".")
		resource := api[len(api)-1]
		if strings.HasSuffix(resource, "List") {
			continue
		}
		resource = strings.ToLower(resource)
		keys = append(keys, resource)
	}
	return keys
}

// DecorateTable takes a table and attempts to add label columns and the
// namespace column. It will fill empty columns with nil (if the object
// does not expose metadata). It returns an error if the table cannot
// be decorated.
func DecorateTable(table *metav1beta1.Table, options PrintOptions) error {
	width := len(table.ColumnDefinitions) + len(options.ColumnLabels)
	if options.WithNamespace {
		width++
	}
	if options.ShowLabels {
		width++
	}

	columns := table.ColumnDefinitions

	nameColumn := -1
	if options.WithKind && !options.Kind.Empty() {
		for i := range columns {
			if columns[i].Format == "name" && columns[i].Type == "string" {
				nameColumn = i
				break
			}
		}
	}

	if width != len(table.ColumnDefinitions) {
		columns = make([]metav1beta1.TableColumnDefinition, 0, width)
		if options.WithNamespace {
			columns = append(columns, metav1beta1.TableColumnDefinition{
				Name: "Namespace",
				Type: "string",
			})
		}
		columns = append(columns, table.ColumnDefinitions...)
		for _, label := range formatLabelHeaders(options.ColumnLabels) {
			columns = append(columns, metav1beta1.TableColumnDefinition{
				Name: label,
				Type: "string",
			})
		}
		if options.ShowLabels {
			columns = append(columns, metav1beta1.TableColumnDefinition{
				Name: "Labels",
				Type: "string",
			})
		}
	}

	rows := table.Rows

	includeLabels := len(options.ColumnLabels) > 0 || options.ShowLabels
	if includeLabels || options.WithNamespace || nameColumn != -1 {
		for i := range rows {
			row := rows[i]

			if nameColumn != -1 {
				row.Cells[nameColumn] = fmt.Sprintf("%s/%s", strings.ToLower(options.Kind.String()), row.Cells[nameColumn])
			}

			var m metav1.Object
			if obj := row.Object.Object; obj != nil {
				if acc, err := meta.Accessor(obj); err == nil {
					m = acc
				}
			}
			// if we can't get an accessor, fill out the appropriate columns with empty spaces
			if m == nil {
				if options.WithNamespace {
					r := make([]interface{}, 1, width)
					row.Cells = append(r, row.Cells...)
				}
				for j := 0; j < width-len(row.Cells); j++ {
					row.Cells = append(row.Cells, nil)
				}
				rows[i] = row
				continue
			}

			if options.WithNamespace {
				r := make([]interface{}, 1, width)
				r[0] = m.GetNamespace()
				row.Cells = append(r, row.Cells...)
			}
			if includeLabels {
				row.Cells = appendLabelCells(row.Cells, m.GetLabels(), options)
			}
			rows[i] = row
		}
	}

	table.ColumnDefinitions = columns
	table.Rows = rows
	return nil
}

func (h *HumanReadablePrinter) printWorkResult(cluster string, raw []byte) *metav1beta1.Table {
	table := &metav1beta1.Table{}
	err := json.Unmarshal(raw, table)
	if err != nil || len(table.ColumnDefinitions) == 0 {
		table = printUnstructured(raw)
		if table == nil {
			return nil
		}
	}

	width := len(table.ColumnDefinitions) + 1
	columns := make([]metav1beta1.TableColumnDefinition, 0, width)
	columns = append(columns, metav1beta1.TableColumnDefinition{
		Name: "Cluster",
		Type: "string",
	})
	table.ColumnDefinitions = append(columns, table.ColumnDefinitions...)

	for i := range table.Rows {
		r := make([]interface{}, 1, width)
		r[0] = cluster
		table.Rows[i].Cells = append(r, table.Rows[i].Cells...)
		if table.Rows[i].Object.Object != nil {
			continue
		}

		partial := &metav1beta1.PartialObjectMetadata{}
		err = json.Unmarshal(table.Rows[i].Object.Raw, partial)
		if err != nil {
			return nil
		}
		table.Rows[i].Object.Object = partial
	}

	return table
}

func (h *HumanReadablePrinter) printWorkResults(obj runtime.Object) *metav1beta1.Table {
	if view, ok := obj.(*mcm.ResourceView); ok {
		if len(view.Status.Conditions) == 0 {
			return nil
		}

		if !view.Spec.SummaryOnly {
			return nil
		}

		listTable := []metav1beta1.Table{}
		returnColumnDefinitions := []metav1beta1.TableColumnDefinition{}
		for cluster, data := range view.Status.Results {
			table := h.printWorkResult(cluster, data.Raw)
			if table == nil {
				return nil
			}
			if len(returnColumnDefinitions) < len(table.ColumnDefinitions) {
				returnColumnDefinitions = table.ColumnDefinitions
			}
			listTable = append(listTable, *table)
		}

		returnTable := &metav1beta1.Table{
			ColumnDefinitions: returnColumnDefinitions,
			Rows:              []metav1beta1.TableRow{},
		}
		for _, table := range listTable {
			formatRow := h.formatTable(returnColumnDefinitions, &table)
			returnTable.Rows = append(returnTable.Rows, formatRow...)
		}
		return returnTable
	}

	return nil
}

//For different api version, the return table is different. so we need to make them consistent
func (h *HumanReadablePrinter) formatTable(column []metav1beta1.TableColumnDefinition, table *metav1beta1.Table) []metav1beta1.TableRow {
	if h.columnSliceEqual(column, table.ColumnDefinitions) {
		return table.Rows
	}
	columnMap := map[string]int{}
	for i, c := range column {
		columnMap[c.Name] = i
	}

	returnTableRow := []metav1beta1.TableRow{}
	for _, tableRow := range table.Rows {
		desCell := make([]interface{}, len(column))
		for i, tcd := range table.ColumnDefinitions {
			if _, ok := columnMap[tcd.Name]; ok {
				desCell[columnMap[tcd.Name]] = tableRow.Cells[i]
			}
		}
		for i, de := range desCell {
			if de == nil {
				desCell[i] = "<none>"
			}
		}
		tableRow.Cells = desCell
		returnTableRow = append(returnTableRow, tableRow)
	}
	return returnTableRow
}

func (h *HumanReadablePrinter) columnSliceEqual(a, b []metav1beta1.TableColumnDefinition) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	for i, v := range a {
		if v.Name != b[i].Name {
			return false
		}
	}
	return true
}

// PrintTable returns a table for the provided object, using the printer registered for that type. It returns
// a table that includes all of the information requested by options, but will not remove rows or columns. The
// caller is responsible for applying rules related to filtering rows or columns.
func (h *HumanReadablePrinter) PrintTable(obj runtime.Object, options PrintOptions) (*metav1beta1.Table, error) {
	resultTable := h.printWorkResults(obj)
	if resultTable != nil {
		return resultTable, nil
	}

	t := reflect.TypeOf(obj)
	handler, ok := h.handlerMap[t]
	if !ok {
		return nil, fmt.Errorf("no table handler registered for this type %v", t)
	}
	if !handler.printRows {
		return h.legacyPrinterToTable(obj, handler)
	}

	args := []reflect.Value{reflect.ValueOf(obj), reflect.ValueOf(options)}
	results := handler.printFunc.Call(args)
	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}

	columns := handler.columnDefinitions
	if !options.Wide {
		columns = make([]metav1beta1.TableColumnDefinition, 0, len(handler.columnDefinitions))
		for i := range handler.columnDefinitions {
			if handler.columnDefinitions[i].Priority != 0 {
				continue
			}
			columns = append(columns, handler.columnDefinitions[i])
		}
	}
	table := &metav1beta1.Table{
		ListMeta: metav1.ListMeta{
			ResourceVersion: "",
		},
		ColumnDefinitions: columns,
		Rows:              results[0].Interface().([]metav1beta1.TableRow),
	}
	if m, err := meta.ListAccessor(obj); err == nil {
		table.ResourceVersion = m.GetResourceVersion()
		table.SelfLink = m.GetSelfLink()
		table.Continue = m.GetContinue()
	} else if m, err := meta.CommonAccessor(obj); err == nil {
		table.ResourceVersion = m.GetResourceVersion()
		table.SelfLink = m.GetSelfLink()
	}
	if err := DecorateTable(table, options); err != nil {
		return nil, err
	}
	return table, nil
}

// legacyPrinterToTable uses the old printFunc with tabbed writer to generate a table.
// TODO: remove when all legacy printers are removed.
func (h *HumanReadablePrinter) legacyPrinterToTable(obj runtime.Object, handler *handlerEntry) (*metav1beta1.Table, error) {
	printFunc := handler.printFunc
	table := &metav1beta1.Table{
		ColumnDefinitions: handler.columnDefinitions,
	}

	options := PrintOptions{
		NoHeaders: true,
		Wide:      true,
	}
	buf := &bytes.Buffer{}
	args := []reflect.Value{reflect.ValueOf(obj), reflect.ValueOf(buf), reflect.ValueOf(options)}

	if meta.IsListType(obj) {
		listInterface, ok := obj.(metav1.ListInterface)
		if ok {
			table.ListMeta.SelfLink = listInterface.GetSelfLink()
			table.ListMeta.ResourceVersion = listInterface.GetResourceVersion()
			table.ListMeta.Continue = listInterface.GetContinue()
		}

		// TODO: this uses more memory than it has to, as we refactor printers we should remove the need
		// for this.
		args[0] = reflect.ValueOf(obj)
		resultValue := printFunc.Call(args)[0]
		if !resultValue.IsNil() {
			return nil, resultValue.Interface().(error)
		}
		data := buf.Bytes()
		i := 0
		items, err := meta.ExtractList(obj)
		if err != nil {
			return nil, err
		}
		for len(data) > 0 {
			cells, remainder := tabbedLineToCells(data, len(table.ColumnDefinitions))
			table.Rows = append(table.Rows, metav1beta1.TableRow{
				Cells:  cells,
				Object: runtime.RawExtension{Object: items[i]},
			})
			data = remainder
			i++
		}
	} else {
		args[0] = reflect.ValueOf(obj)
		resultValue := printFunc.Call(args)[0]
		if !resultValue.IsNil() {
			return nil, resultValue.Interface().(error)
		}
		data := buf.Bytes()
		cells, _ := tabbedLineToCells(data, len(table.ColumnDefinitions))
		table.Rows = append(table.Rows, metav1beta1.TableRow{
			Cells:  cells,
			Object: runtime.RawExtension{Object: obj},
		})
	}
	return table, nil
}

func formatLabelHeaders(columnLabels []string) []string {
	formHead := make([]string, len(columnLabels))
	for i, l := range columnLabels {
		p := strings.Split(l, "/")
		formHead[i] = strings.ToUpper((p[len(p)-1]))
	}
	return formHead
}

// appendLabelCells returns a slice of value columns matching the requested print options.
// Intended for use with tables.
func appendLabelCells(values []interface{}, itemLabels map[string]string, opts PrintOptions) []interface{} {
	for _, key := range opts.ColumnLabels {
		values = append(values, itemLabels[key])
	}
	if opts.ShowLabels {
		values = append(values, labels.FormatLabels(itemLabels))
	}
	return values
}

// FormatResourceName receives a resource kind, name, and boolean specifying
// whether or not to update the current name to "kind/name"
func FormatResourceName(kind schema.GroupKind, name string, withKind bool) string {
	if !withKind || kind.Empty() {
		return name
	}

	return strings.ToLower(kind.String()) + "/" + name
}

func AppendLabels(itemLabels map[string]string, columnLabels []string) string {
	var buffer bytes.Buffer

	for _, cl := range columnLabels {
		buffer.WriteString(fmt.Sprint("\t"))
		if il, ok := itemLabels[cl]; ok {
			buffer.WriteString(fmt.Sprint(il))
		} else {
			buffer.WriteString("<none>")
		}
	}

	return buffer.String()
}

// Append all labels to a single column. We need this even when show-labels flag* is
// false, since this adds newline delimiter to the end of each row.
func AppendAllLabels(showLabels bool, itemLabels map[string]string) string {
	var buffer bytes.Buffer

	if showLabels {
		buffer.WriteString(fmt.Sprint("\t"))
		buffer.WriteString(labels.FormatLabels(itemLabels))
	}
	buffer.WriteString("\n")

	return buffer.String()
}

func tabbedLineToCells(data []byte, expected int) ([]interface{}, []byte) {
	var remainder []byte
	max := bytes.Index(data, []byte("\n"))
	if max != -1 {
		remainder = data[max+1:]
		data = data[:max]
	}
	cells := make([]interface{}, expected)
	for i := 0; i < expected; i++ {
		next := bytes.Index(data, []byte("\t"))
		if next == -1 {
			cells[i] = string(data)
			// fill the remainder with empty strings, this indicates a printer bug
			for j := i + 1; j < expected; j++ {
				cells[j] = ""
			}
			break
		}
		cells[i] = string(data[:next])
		data = data[next+1:]
	}
	return cells, remainder
}

func printUnstructured(raw []byte) *metav1beta1.Table {
	objectMetaColumnDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}

	obj := &unstructured.Unstructured{}
	err := json.Unmarshal(raw, obj)
	if err != nil {
		return nil
	}

	rows, err := printObjectMeta(obj)
	if err != nil {
		return nil
	}

	return &metav1beta1.Table{
		ColumnDefinitions: objectMetaColumnDefinitions,
		Rows:              rows,
	}
}

func printObjectMeta(obj runtime.Object) ([]metav1beta1.TableRow, error) {
	if meta.IsListType(obj) {
		rows := make([]metav1beta1.TableRow, 0, 16)
		err := meta.EachListItem(obj, func(obj runtime.Object) error {
			nestedRows, err := printObjectMeta(obj)
			if err != nil {
				return err
			}
			rows = append(rows, nestedRows...)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return rows, nil
	}

	rows := make([]metav1beta1.TableRow, 0, 1)
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}
	row.Cells = append(row.Cells, m.GetName(), translateTimestamp(m.GetCreationTimestamp()))
	rows = append(rows, row)
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
