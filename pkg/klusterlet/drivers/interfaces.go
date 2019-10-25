// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package drivers

import (
	"io"
	"net/url"
)

// LogInterface is the interface to provide log
type LogInterface interface {
	// GetContainerLog read log of a certain container
	GetContainerLog(namespace, podID, containerName string, query url.Values, stdout io.Writer) error
}

// MonitorInterface is the interface to provide monitoring data
type MonitorInterface interface {
	GetMetrics(queryPath string, query url.Values, stdout io.Writer) error
}
