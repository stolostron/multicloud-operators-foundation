// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package internalversion

import (
	"strings"
	"time"

	hcm "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	hcmv1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	clusterregistryv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/klog"
)

// AddHandlers adds print handlers for default Kubernetes types dealing with internal versions.
func AddHandlers(h printers.PrintHandler) {
	clusterColumnDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Endpoints", Type: "string"},
		{Name: "Status", Type: "string"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	addTableHandler(h, clusterColumnDefinitions, printCluster)
	addTableHandler(h, clusterColumnDefinitions, printClusterList)

	clusterStatusColumnDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Addresses", Type: "string"},
		{Name: "Used/Total CPU", Type: "string"},
		{Name: "Used/Total Memory", Type: "string"},
		{Name: "Used/Total Storage", Type: "string"},
		{Name: "Node", Type: "string"},
		{Name: "Pod", Type: "string"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Version", Type: "string"},
		{Name: "KlusterletVersion", Type: "string"},
		{Name: "EndpointVersion", Type: "string"},
		{Name: "EndpointOperatorVersion", Type: "string"},
	}
	addTableHandler(h, clusterStatusColumnDefinitions, printClusterStatus)
	addTableHandler(h, clusterStatusColumnDefinitions, printClusterStatusList)

	worksetColumnDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Cluster selector", Type: "string"},
		{Name: "Status", Type: "string"},
		{Name: "Reason", Type: "string"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	addTableHandler(h, worksetColumnDefinitions, printWorkSet)
	addTableHandler(h, worksetColumnDefinitions, printWorkSetList)

	resourceviewColumnDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Cluster selector", Type: "string"},
		{Name: "Status", Type: "string"},
		{Name: "Reason", Type: "string"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	addTableHandler(h, resourceviewColumnDefinitions, printResourceView)
	addTableHandler(h, resourceviewColumnDefinitions, printResourceViewList)

	workColumnDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Type", Type: "string"},
		{Name: "Cluster", Type: "string"},
		{Name: "Status", Type: "string"},
		{Name: "Reason", Type: "string"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	addTableHandler(h, workColumnDefinitions, printWork)
	addTableHandler(h, workColumnDefinitions, printWorkList)

	hcmJoinDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Cluster Name", Type: "string"},
		{Name: "Cluster Namespace", Type: "string"},
		{Name: "Status", Type: "string"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	addTableHandler(h, hcmJoinDefinitions, printHCMJoin)
	addTableHandler(h, hcmJoinDefinitions, printHCMJoinList)

	AddDefaultHandlers(h)
}

// AddDefaultHandlers adds handlers that can work with most Kubernetes objects.
func AddDefaultHandlers(h printers.PrintHandler) {
	// types without defined columns
	objectMetaColumnDefinitions := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	if err := h.DefaultTableHandler(objectMetaColumnDefinitions, printObjectMeta); err != nil {
		klog.Errorf("failed to default print handler with a given set of columns, %v", err)
	}
}

func printObjectMeta(obj runtime.Object, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	if meta.IsListType(obj) {
		rows := make([]metav1beta1.TableRow, 0, 16)
		err := meta.EachListItem(obj, func(obj runtime.Object) error {
			nestedRows, err := printObjectMeta(obj, options)
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

func printClusterList(list *clusterregistryv1alpha1.ClusterList, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printCluster(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printCluster(obj *clusterregistryv1alpha1.Cluster, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}

	endpoints := convertClusterEndPoints(obj.Spec.KubernetesAPIEndpoints)
	status := convertClusterStatus(obj.Status.Conditions)
	row.Cells = append(row.Cells, obj.Name, endpoints, status, translateTimestamp(obj.CreationTimestamp))
	return []metav1beta1.TableRow{row}, nil
}

func convertClusterStatus(conditions []clusterregistryv1alpha1.ClusterCondition) string {
	status := ""
	if len(conditions) == 0 {
		status = "Pending"
	} else {
		condition := conditions[len(conditions)-1]
		if condition.Type == clusterregistryv1alpha1.ClusterOK {
			status = "Ready"
		} else {
			status = "Offline"
		}
	}

	return status
}

func convertClusterEndPoints(endpoints clusterregistryv1alpha1.KubernetesAPIEndpoints) string {
	endpointStr := []string{}
	for _, endpoint := range endpoints.ServerEndpoints {
		endpointStr = append(endpointStr, endpoint.ServerAddress)
	}

	return strings.Join(endpointStr, ",")
}

func printClusterStatusList(list *hcm.ClusterStatusList, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printClusterStatus(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printClusterStatus(obj *hcm.ClusterStatus, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}

	cpu := ""
	if cpudata, ok := obj.Spec.Capacity[apiv1.ResourceCPU]; ok {
		cpu = cpudata.String()
	}

	usedcpu := ""
	if usedcpudata, ok := obj.Spec.Usage[apiv1.ResourceCPU]; ok {
		usedcpu = usedcpudata.String()
	}

	memory := ""
	if memdata, ok := obj.Spec.Capacity[apiv1.ResourceMemory]; ok {
		memory = memdata.String()
	}
	usedmemory := ""
	if usedmemdata, ok := obj.Spec.Usage[apiv1.ResourceMemory]; ok {
		usedmemory = usedmemdata.String()
	}

	storage := ""
	if storagedata, ok := obj.Spec.Capacity[apiv1.ResourceStorage]; ok {
		storage = storagedata.String()
	}
	usedstorage := ""
	if usedstoragedata, ok := obj.Spec.Usage[apiv1.ResourceStorage]; ok {
		usedstorage = usedstoragedata.String()
	}

	pod := ""
	if poddata, ok := obj.Spec.Usage[hcmv1alpha1.ResourcePods]; ok {
		pod = poddata.String()
	}

	node := ""
	if nodedata, ok := obj.Spec.Capacity[hcmv1alpha1.ResourceNodes]; ok {
		node = nodedata.String()
	}

	version := obj.Spec.KlusterletVersion
	serverVersion := obj.Spec.Version
	endpointVers := obj.Spec.EndpointVersion
	endpointOptVers := obj.Spec.EndpointOperatorVersion

	addresses := convertEndpointAddresses(obj.Spec.MasterAddresses)

	row.Cells = append(row.Cells, obj.Name, addresses, usedcpu+"/"+cpu, usedmemory+"/"+memory, usedstorage+"/"+storage, node, pod,
		translateTimestamp(obj.CreationTimestamp), serverVersion, version, endpointVers, endpointOptVers)
	return []metav1beta1.TableRow{row}, nil
}

func convertEndpointAddresses(endpointAddresses []apiv1.EndpointAddress) string {
	addresses := []string{}
	for _, endpointAddress := range endpointAddresses {
		if endpointAddress.Hostname != "" {
			addresses = append(addresses, endpointAddress.Hostname)
		} else if endpointAddress.IP != "" {
			addresses = append(addresses, endpointAddress.IP)
		}
	}

	return strings.Join(addresses, ",")
}

func printWorkSetList(list *hcm.WorkSetList, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printWorkSet(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printWorkSet(obj *hcm.WorkSet, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}

	row.Cells = append(row.Cells, obj.Name, metav1.FormatLabelSelector(obj.Spec.ClusterSelector), string(obj.Status.Status), obj.Status.Reason,
		translateTimestamp(obj.CreationTimestamp))
	return []metav1beta1.TableRow{row}, nil
}

func printResourceViewList(list *hcm.ResourceViewList, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printResourceView(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printResourceView(obj *hcm.ResourceView, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}

	status := ""
	reason := ""
	if len(obj.Status.Conditions) > 0 {
		status = string(obj.Status.Conditions[len(obj.Status.Conditions)-1].Type)
		reason = obj.Status.Conditions[len(obj.Status.Conditions)-1].Reason
	}

	row.Cells = append(row.Cells, obj.Name, metav1.FormatLabelSelector(obj.Spec.ClusterSelector), status, reason,
		translateTimestamp(obj.CreationTimestamp))
	return []metav1beta1.TableRow{row}, nil
}

func printWorkList(list *hcm.WorkList, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printWork(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printWork(obj *hcm.Work, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}

	row.Cells = append(row.Cells, obj.Name, obj.Spec.Type, obj.Spec.Cluster.Name, string(obj.Status.Type), obj.Status.Reason,
		translateTimestamp(obj.CreationTimestamp))
	return []metav1beta1.TableRow{row}, nil
}

func printHCMJoinList(list *hcm.ClusterJoinRequestList, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printHCMJoin(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printHCMJoin(obj *hcm.ClusterJoinRequest, options printers.PrintOptions) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}
	joinStatus := string(obj.Status.Phase)
	if len(joinStatus) == 0 {
		joinStatus = "Pending"
	}
	row.Cells = append(row.Cells, obj.Name, obj.Spec.ClusterName, obj.Spec.ClusterNamespace, joinStatus,
		translateTimestamp(obj.CreationTimestamp))
	return []metav1beta1.TableRow{row}, nil
}

func addTableHandler(h printers.PrintHandler, columnDefinitions []metav1beta1.TableColumnDefinition, printFunc interface{}) {
	if err := h.TableHandler(columnDefinitions, printFunc); err != nil {
		klog.Errorf("failed to add print handler with a given set of columns, %v", err)
	}
}
