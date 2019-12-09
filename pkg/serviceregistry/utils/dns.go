// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/plugin"
)

// UpdateSOARecord updates SOA record from old DNS records
func UpdateSOARecord(oldRecords string, updateTime int64) (string, string, error) {
	records := strings.Split(oldRecords, "\n")
	if len(records) == 0 {
		return "", "", fmt.Errorf("no dns records")
	}
	old := records[0]
	soaParts := regexp.MustCompile(`\s+`).Split(old, -1)
	//e.g. mcm.svc.    IN    SOA    mcmcoredns.kube-system.svc.cluster.local. host.cluster.local. 1544779186 7200 1800 86400 30
	if len(soaParts) != 10 || soaParts[2] != "SOA" {
		return "", "", fmt.Errorf("wrong format")
	}
	return old,
		fmt.Sprintf("%s\tIN\tSOA\t%s %s %d %s %s %s %s",
			soaParts[0],
			soaParts[3], soaParts[4], updateTime, soaParts[6], soaParts[7], soaParts[8], soaParts[9]),
		nil
}

// NewServiceDNSRecords returns service DNS records form service locations
func NewServiceDNSRecords(currentClusterInfo plugin.ClusterInfo,
	dnsSuffix string,
	serviceLocations []*plugin.ServiceLocation) string {
	// cache the services that have the same host domain
	locationsMap := make(map[string][]*plugin.ServiceLocation)
	records := ""

	for _, location := range serviceLocations {
		mainHost := location.Hosts[0]
		clusterDomain := toDomainName(mainHost, dnsSuffix, location.ClusterInfo.Name)
		records += toClusterDomainRecord(clusterDomain, location)
		// a service has many hosts, e.g. ingress
		for _, host := range location.Hosts[1:] {
			records += toHostDomainRecord(toDomainName(host, dnsSuffix, ""), clusterDomain)
		}
		locations, ok := locationsMap[mainHost]
		if !ok {
			locationsMap[mainHost] = []*plugin.ServiceLocation{location}
			continue
		}
		locationsMap[mainHost] = append(locations, location)
	}

	// handle main domain
	for mainHost, locations := range locationsMap {
		nearestLocation := findNearestLocation(locations, currentClusterInfo)
		nearestClusterDomain := toDomainName(mainHost, dnsSuffix, nearestLocation.ClusterInfo.Name)
		records += toHostDomainRecord(toDomainName(mainHost, dnsSuffix, ""), nearestClusterDomain)
	}
	return records
}

// NeedToUpdateDNSRecords compares two dns records to determine update is needed or not
func NeedToUpdateDNSRecords(oldDNSRecords, newDNSRecords string) bool {
	oldRecords := strings.Split(oldDNSRecords, "\n")
	newRecords := strings.Split(newDNSRecords, "\n")
	sort.Strings(oldRecords)
	sort.Strings(newRecords)
	return fmt.Sprintf("%s", oldRecords) != fmt.Sprintf("%s", newRecords)
}

func findNearestLocation(locations []*plugin.ServiceLocation, currentClusterInfo plugin.ClusterInfo) *plugin.ServiceLocation {
	nearest := locations[0]
	for _, location := range locations {
		// same cluster, the nearest
		if location.ClusterInfo.Name == currentClusterInfo.Name {
			nearest = location
			break
		}
		// zone is nearer than region
		if location.ClusterInfo.Region == currentClusterInfo.Region {
			nearest = location
			if location.ClusterInfo.Zone == currentClusterInfo.Zone {
				nearest = location
			}
		}
	}
	return nearest
}

func toDomainName(hostName, suffix, clusterName string) string {
	if clusterName != "" {
		return fmt.Sprintf("%s.%s.%s.", strings.TrimSuffix(hostName, "."+suffix), clusterName, suffix)
	}
	return fmt.Sprintf("%s.%s.", strings.TrimSuffix(hostName, "."+suffix), suffix)
}

func toClusterDomainRecord(domain string, location *plugin.ServiceLocation) string {
	if location.Address.IP != "" {
		return fmt.Sprintf("%s\tIN\tA\t%s\n", domain, location.Address.IP)
	}
	return fmt.Sprintf("%s\tIN\tCNAME\t%s\n", domain, location.Address.Hostname)
}

func toHostDomainRecord(hostDomain, clusterDomain string) string {
	return fmt.Sprintf("%s\tIN\tCNAME\t%s\n", hostDomain, clusterDomain)
}
