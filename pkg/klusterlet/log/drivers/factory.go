package drivers

import (
	"io"
	"net/url"

	"k8s.io/client-go/kubernetes"
)

// DriverFactory is factory to install all drivers
type DriverFactory struct {
	logDriver LogInterface
}

// LogInterface is the interface to provide log
type LogInterface interface {
	// GetContainerLog read log of a certain container
	GetContainerLog(namespace, podID, containerName string, query url.Values, stdout io.Writer) error
}

func NewDriverFactory(kubeclient kubernetes.Interface) *DriverFactory {
	return &DriverFactory{
		logDriver: NewLogDriver(kubeclient),
	}
}

func (d *DriverFactory) LogDriver() LogInterface {
	return d.logDriver
}

func (d *DriverFactory) SetLogDriver(logDriver LogInterface) {
	d.logDriver = logDriver
}
