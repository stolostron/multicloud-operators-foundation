package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func FormatQuatityToMi(q resource.Quantity) resource.Quantity {
	raw, _ := q.AsInt64()
	raw /= (1024 * 1024)
	rq, err := resource.ParseQuantity(fmt.Sprintf("%dMi", raw))
	if err != nil {
		return q
	}
	return rq
}

func FormatQuatityToGi(q resource.Quantity) resource.Quantity {
	raw, _ := q.AsInt64()
	raw /= (1024 * 1024 * 1024)
	rq, err := resource.ParseQuantity(fmt.Sprintf("%dGi", raw))
	if err != nil {
		return q
	}
	return rq
}

// PodRequestsAndLimits returns a dictionary of all defined resources summed up for all
// containers of the pod.
func PodRequestsAndLimits(
	pod *corev1.Pod) (reqs map[corev1.ResourceName]resource.Quantity, limits map[corev1.ResourceName]resource.Quantity) {
	reqs, limits = map[corev1.ResourceName]resource.Quantity{}, map[corev1.ResourceName]resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if value, ok := reqs[name]; !ok {
				reqs[name] = quantity.DeepCopy()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = quantity.DeepCopy()
			} else {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		for name, quantity := range container.Resources.Requests {
			value, ok := reqs[name]
			if !ok {
				reqs[name] = quantity.DeepCopy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				reqs[name] = quantity.DeepCopy()
			}
		}
		for name, quantity := range container.Resources.Limits {
			value, ok := limits[name]
			if !ok {
				limits[name] = quantity.DeepCopy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				limits[name] = quantity.DeepCopy()
			}
		}
	}
	return
}

func GetCPUAndMemoryCapacity(nodes []*corev1.Node) (cpuCapacity, memoryCapacity resource.Quantity) {
	cpuCapacity = *resource.NewQuantity(int64(0), resource.DecimalSI)
	memoryCapacity = *resource.NewQuantity(int64(0), resource.BinarySI)
	for _, node := range nodes {
		cpuCapacity.Add(*node.Status.Capacity.Cpu())
		memoryCapacity.Add(*node.Status.Capacity.Memory())
	}
	return cpuCapacity, memoryCapacity
}

func GetStorageCapacityAndAllocation(pvs []*corev1.PersistentVolume) (storageCapacity, storageAllocation resource.Quantity) {
	storageCapacity = *resource.NewQuantity(int64(0), resource.BinarySI)
	storageAllocation = *resource.NewQuantity(int64(0), resource.BinarySI)
	for _, pv := range pvs {
		storageCapacity.Add(pv.Spec.Capacity["storage"])
		if pv.Status.Phase != "Available" {
			storageAllocation.Add(pv.Spec.Capacity["storage"])
		}
	}
	return storageCapacity, storageAllocation
}

func GetCPUAndMemoryAllocation(pods []*corev1.Pod) (cpuAllocation, memoryAllocation resource.Quantity) {
	cpuAllocation = *resource.NewQuantity(int64(0), resource.DecimalSI)
	memoryAllocation = *resource.NewQuantity(int64(0), resource.BinarySI)
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodPending {
			continue
		}
		podReqs, _ := PodRequestsAndLimits(pod)

		for podReqName, podReqValue := range podReqs {
			if podReqName == corev1.ResourceCPU {
				cpuAllocation.Add(podReqValue)
			} else if podReqName == corev1.ResourceMemory {
				memoryAllocation.Add(podReqValue)
			}
		}
	}
	return cpuAllocation, memoryAllocation
}
