package api

import (
	"context"
	"testing"

	proxyv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/proxy/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/printers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/printers/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestClusterStatusStorage() *clusterStatusStorage {
	return &clusterStatusStorage{
		tableConverter: storage.TableConvertor{
			TableGenerator: printers.NewTableGenerator().With(addClusterStatusHandlers),
		},
	}
}

func TestConvertToTable_ClusterStatus(t *testing.T) {
	s := newTestClusterStatusStorage()
	obj := &proxyv1beta1.ClusterStatus{
		ObjectMeta: metav1.ObjectMeta{Name: "spoke-1"},
	}

	table, err := s.ConvertToTable(context.TODO(), obj, &metav1.TableOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if table == nil {
		t.Fatal("expected non-nil table")
	}
	if len(table.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(table.Rows))
	}
	if table.Rows[0].Cells[0] != "spoke-1" {
		t.Errorf("expected cell value 'spoke-1', got %v", table.Rows[0].Cells[0])
	}
}

func TestConvertToTable_ClusterStatusList(t *testing.T) {
	s := newTestClusterStatusStorage()
	obj := &proxyv1beta1.ClusterStatusList{
		Items: []proxyv1beta1.ClusterStatus{
			{ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"}},
		},
	}

	table, err := s.ConvertToTable(context.TODO(), obj, &metav1.TableOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if table == nil {
		t.Fatal("expected non-nil table")
	}
	if len(table.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table.Rows))
	}
	if table.Rows[0].Cells[0] != "cluster-a" {
		t.Errorf("expected cell value 'cluster-a', got %v", table.Rows[0].Cells[0])
	}
	if table.Rows[1].Cells[0] != "cluster-b" {
		t.Errorf("expected cell value 'cluster-b', got %v", table.Rows[1].Cells[0])
	}
}

func TestConvertToTable_EmptyList(t *testing.T) {
	s := newTestClusterStatusStorage()
	obj := &proxyv1beta1.ClusterStatusList{}

	table, err := s.ConvertToTable(context.TODO(), obj, &metav1.TableOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if table == nil {
		t.Fatal("expected non-nil table")
	}
	if len(table.Rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(table.Rows))
	}
}
