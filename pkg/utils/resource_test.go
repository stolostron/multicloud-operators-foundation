package utils

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFormatQuatityToMi(t *testing.T) {
	type args struct {
		q resource.Quantity
	}
	m1 := *resource.NewQuantity(int64(1024*1024), resource.BinarySI)
	m2 := *resource.NewQuantity(int64(1024*1024*2), resource.BinarySI)
	tests := []struct {
		name string
		args args
		want string
	}{
		{"case1:", args{m1}, "1Mi"},
		{"case1:", args{m2}, "2Mi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatQuatityToMi(tt.args.q); !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("FormatQuatityToMi() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestFormatQuatityToGi(t *testing.T) {
	type args struct {
		q resource.Quantity
	}
	m1 := *resource.NewQuantity(int64(1024*1024*1024), resource.BinarySI)
	m2 := *resource.NewQuantity(int64(1024*1024*1024*2), resource.BinarySI)
	tests := []struct {
		name string
		args args
		want string
	}{
		{"case1:", args{m1}, "1Gi"},
		{"case1:", args{m2}, "2Gi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatQuatityToGi(tt.args.q); !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("FormatQuatityToMi() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCPUAndMemoryCapacity(t *testing.T) {
	nodeMemory := *resource.NewQuantity(int64(1024*1024*1024), resource.BinarySI)
	nodes := []*corev1.Node{
		{
			Status: corev1.NodeStatus{
				Capacity: corev1.ResourceList{
					corev1.ResourceMemory: nodeMemory,
				},
			},
		},
	}
	cpu, memory := GetCPUAndMemoryCapacity(nodes)
	if toIntQuantity(cpu) != 0 {
		t.Errorf("Failed to get CPU capacity")
	}
	if toIntQuantity(memory) != int64(1024*1024*1024) {
		t.Errorf("Failed to get memory capacity")
	}
}

func TestGetStorageCapacityAndAllocation(t *testing.T) {
	storageCapacity := *resource.NewQuantity(int64(1024*1024*1024), resource.BinarySI)
	pvcs := []*corev1.PersistentVolume{
		{Spec: corev1.PersistentVolumeSpec{Capacity: corev1.ResourceList{"storage": storageCapacity}}},
	}
	capacity, allocation := GetStorageCapacityAndAllocation(pvcs)
	if toIntQuantity(capacity) != int64(1024*1024*1024) {
		t.Errorf("Failed to get storage capacity")
	}
	if toIntQuantity(allocation) != int64(1024*1024*1024) {
		t.Errorf("Failed to get storage allocation")
	}
}

func TestGetCPUAndMemoryAllocation(t *testing.T) {
	pods := []*corev1.Pod{
		{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{corev1.ResourceCPU: *resource.NewQuantity(int64(16), resource.BinarySI)},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewQuantity(int64(16), resource.BinarySI),
							corev1.ResourceMemory: *resource.NewQuantity(int64(1024), resource.BinarySI),
						},
					}},
				},
				InitContainers: []corev1.Container{
					{Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{corev1.ResourceCPU: *resource.NewQuantity(int64(16), resource.BinarySI)},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    *resource.NewQuantity(int64(16), resource.BinarySI),
							corev1.ResourceMemory: *resource.NewQuantity(int64(1024), resource.BinarySI),
						},
					}},
				},
			},
		},
	}
	cpuAllocation, memoryAllocation := GetCPUAndMemoryAllocation(pods)
	if toIntQuantity(cpuAllocation) != 16 {
		t.Errorf("Failed to get cpu allocation")
	}
	if toIntQuantity(memoryAllocation) != 1024 {
		t.Errorf("Failed to get memory allocation")
	}
}

func toIntQuantity(quantity resource.Quantity) int64 {
	value, ok := quantity.AsInt64()
	if !ok {
		return 0
	}
	return value
}
