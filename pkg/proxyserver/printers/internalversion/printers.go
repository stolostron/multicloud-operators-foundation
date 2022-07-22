package internalversion

import (
	"strings"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/printers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/klog/v2"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func AddHandlers(h printers.PrintHandler) {
	managedClusterColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: "Name is the name of managedCluster."},
		{Name: "Hub Accepted", Type: "boolean", Description: "Hub Accepted represents that hub accepts the joining of Klusterlet agent on the managed cluster with the hub."},
		{Name: "Managed Cluster URLs", Type: "string", Description: "Managed Cluster URLs is the URL of apiserver endpoint of the managed cluster."},
		{Name: "Joined", Type: "string", Description: "Joined represents the managed cluster has successfully joined the hub."},
		{Name: "Available", Type: "string", Description: "Available represents the managed cluster is available."},
		{Name: "Age", Type: "date", Description: "Age represents the age of the managedCluster until created."},
	}
	err := h.TableHandler(managedClusterColumnDefinitions, printManagedCluster)
	if err != nil {
		klog.Warningf("%v", err)
	}
	err = h.TableHandler(managedClusterColumnDefinitions, printManagedClusterList)
	if err != nil {
		klog.Warningf("%v", err)
	}
}

func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

func printManagedCluster(obj *clusterv1.ManagedCluster, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}
	hubAccepted := obj.Spec.HubAcceptsClient
	urls := []string{}
	for _, config := range obj.Spec.ManagedClusterClientConfigs {
		urls = append(urls, config.URL)
	}
	allUrls := strings.Join(urls, ",")
	joined := ""
	available := ""
	for _, cond := range obj.Status.Conditions {
		if cond.Type == clusterv1.ManagedClusterConditionJoined {
			joined = string(cond.Status)
		}
		if cond.Type == clusterv1.ManagedClusterConditionAvailable {
			available = string(cond.Status)
		}
	}

	age := translateTimestampSince(obj.CreationTimestamp)

	row.Cells = append(row.Cells, obj.Name, hubAccepted, allUrls, joined, available, age)
	return []metav1.TableRow{row}, nil
}

func printManagedClusterList(list *clusterv1.ManagedClusterList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(list.Items))
	for i := range list.Items {
		r, err := printManagedCluster(&list.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}
