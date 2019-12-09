// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/plugin"
)

func TestUpdateSOARecord(t *testing.T) {
	oldRecords := "mcm.svc.\tIN\tSOA\tmcmdns.kube-system.svc.cluster.local. h.cluster.local. 1544779186 7200 1800 86400 30"
	updateTime := time.Now().Unix()
	old, newDNSRecords, _ := UpdateSOARecord(oldRecords, updateTime)
	if old != oldRecords {
		t.Fatalf("Expected to\n %s, but\n %s", oldRecords, old)
	}
	newRecords := fmt.Sprintf(
		"mcm.svc.\tIN\tSOA\tmcmdns.kube-system.svc.cluster.local. h.cluster.local. %d 7200 1800 86400 30", updateTime)
	if newDNSRecords != newRecords {
		t.Fatalf("Expected to\n %s, but\n %s", newRecords, newDNSRecords)
	}
}

func TestNewServiceDNSRecords(t *testing.T) {
	var expected string
	var actual string

	currentClusterInfo := plugin.ClusterInfo{
		Name:   "c1",
		Zone:   "z1",
		Region: "us-east",
	}
	serviceLocations := []*plugin.ServiceLocation{
		newServiceLocation("1.2.3.4", "", []string{"svc.test"}, "c2", "z1", "us-west"),
	}

	// test one service
	expected = "svc.test.c2.mcm.svc.\tIN\tA\t1.2.3.4\n" +
		"svc.test.mcm.svc.\tIN\tCNAME\tsvc.test.c2.mcm.svc.\n"
	actual = NewServiceDNSRecords(currentClusterInfo, "mcm.svc", serviceLocations)
	if actual != expected {
		t.Fatalf("Expected to\n %s, but\n %s", expected, actual)
	}

	// test nearest service
	serviceLocations = append(
		serviceLocations,
		newServiceLocation("5.6.7.8", "", []string{"svc.test"}, "c3", "z2", "us-east"),
		newServiceLocation("5.6.7.9", "", []string{"svc.test"}, "c4", "z1", "us-east"))
	expected = "svc.test.c2.mcm.svc.\tIN\tA\t1.2.3.4\n" +
		"svc.test.c3.mcm.svc.\tIN\tA\t5.6.7.8\n" +
		"svc.test.c4.mcm.svc.\tIN\tA\t5.6.7.9\n" +
		"svc.test.mcm.svc.\tIN\tCNAME\tsvc.test.c4.mcm.svc.\n"
	actual = NewServiceDNSRecords(currentClusterInfo, "mcm.svc", serviceLocations)
	if actual != expected {
		t.Fatalf("Expected to\n %s, but\n %s", expected, actual)
	}

	// test local service
	serviceLocations = append(serviceLocations, newServiceLocation("10.1.2.3", "", []string{"svc.test"}, "c1", "z1", "us-east"))
	expected = "svc.test.c2.mcm.svc.\tIN\tA\t1.2.3.4\n" +
		"svc.test.c3.mcm.svc.\tIN\tA\t5.6.7.8\n" +
		"svc.test.c4.mcm.svc.\tIN\tA\t5.6.7.9\n" +
		"svc.test.c1.mcm.svc.\tIN\tA\t10.1.2.3\n" +
		"svc.test.mcm.svc.\tIN\tCNAME\tsvc.test.c1.mcm.svc.\n"
	actual = NewServiceDNSRecords(currentClusterInfo, "mcm.svc", serviceLocations)
	if actual != expected {
		t.Fatalf("Expected to\n %s, but\n %s", expected, actual)
	}

	// ingress
	serviceLocations = append(serviceLocations,
		newServiceLocation("", "xxxx.us-east-1.aws.com",
			[]string{"ing.test.mcm.svc", "foo.bar.mcm.svc"},
			"c1", "z1", "us-east"))
	expected = "svc.test.c2.mcm.svc.\tIN\tA\t1.2.3.4\n" +
		"svc.test.c3.mcm.svc.\tIN\tA\t5.6.7.8\n" +
		"svc.test.c4.mcm.svc.\tIN\tA\t5.6.7.9\n" +
		"svc.test.c1.mcm.svc.\tIN\tA\t10.1.2.3\n" +
		"ing.test.c1.mcm.svc.\tIN\tCNAME\txxxx.us-east-1.aws.com\n" +
		"foo.bar.mcm.svc.\tIN\tCNAME\ting.test.c1.mcm.svc.\n"
	expectedSvc := "svc.test.mcm.svc.\tIN\tCNAME\tsvc.test.c1.mcm.svc.\n"
	expectedIng := "ing.test.mcm.svc.\tIN\tCNAME\ting.test.c1.mcm.svc.\n"
	actual = NewServiceDNSRecords(currentClusterInfo, "mcm.svc", serviceLocations)
	if strings.Contains(expected, actual) && strings.Contains(expectedSvc, actual) && strings.Contains(expectedIng, actual) {
		t.Fatalf("Expected to\n %s, but\n %s", expected, actual)
	}
}

func TestNeedToUpdateDNSRecords(t *testing.T) {
	old := "mcm.svc.\tIN\tSOA\ttest.kube-system.svc.cluster.local. hostmaster.cluster.local. 1551062619 7200 1800 86400 30\n" +
		"httpbin.test.mylocaltest.mcm.svc.\tIN\tA\t9.197.118.68\n" +
		"httpbin.test.cluster1.mcm.svc.\tIN\tA\t9.197.118.66\n"
	newRecord := "mcm.svc.\tIN\tSOA\ttest.kube-system.svc.cluster.local." +
		" hostmaster.cluster.local. 1551062619 7200 1800 86400 30\n" +
		"httpbin.test.cluster1.mcm.svc.\tIN\tA\t9.197.118.66\n" +
		"httpbin.test.mylocaltest.mcm.svc.\tIN\tA\t9.197.118.68\n"
	result := NeedToUpdateDNSRecords(old, newRecord)
	if result {
		t.Fatalf("Expected to false, but true")
	}
	old = "mcm.svc.\tIN\tSOA\ttest.kube-system.svc.cluster.local. hostmaster.cluster.local. 1551062619 7200 1800 86400 30\n" +
		"httpbin.test.cluster1.mcm.svc.\tIN\tA\t9.197.118.66\n" +
		"httpbin.test.cluster2.mcm.svc.\tIN\tA\t9.197.118.67\n"
	newRecord = "mcm.svc.\tIN\tSOA\ttest.kube-system.svc.cluster.local." +
		" hostmaster.cluster.local. 1551062619 7200 1800 86400 30\n" +
		"httpbin.test.cluster1.mcm.svc.\tIN\tA\t9.197.118.69\n" +
		"httpbin.test.cluster2.mcm.svc.\tIN\tA\t9.197.118.67\n"
	result = NeedToUpdateDNSRecords(old, newRecord)
	if !result {
		t.Fatalf("Expected to true, but fasle")
	}
}

func newServiceLocation(ip, hostname string, hosts []string, clusterName, zone, region string) *plugin.ServiceLocation {
	return &plugin.ServiceLocation{
		Address: plugin.ServiceAddress{
			IP:       ip,
			Hostname: hostname,
		},
		Hosts: hosts,
		ClusterInfo: plugin.ClusterInfo{
			Name:   clusterName,
			Zone:   zone,
			Region: region,
		},
	}
}
