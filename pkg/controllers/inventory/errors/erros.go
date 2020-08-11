package errors

import (
	"fmt"
)

// NoClusterSpecifiedError is an error when no cluster is specified
type NoClusterSpecifiedError struct {
	Message string
}

// Error returns a string representation of the NoClusterSpecifiedError
func (e NoClusterSpecifiedError) Error() string {
	return e.Message
}

// NewNoClusterError returns a NoClusterSpecifiedError
func NewNoClusterError() error {
	return &NoClusterSpecifiedError{
		Message: "No cluster specified",
	}
}

// IsNoClusterError returns true if the err is a NoClusterSpecifiedError
func IsNoClusterError(err error) bool {
	_, ok := err.(*NoClusterSpecifiedError)
	return ok
}

// AssetSecretNotFoundError is an error when the asset's secret can not be found
type AssetSecretNotFoundError struct {
	Name      string
	Namespace string
}

// Error returns a string representation of the AssetSecretNotFoundError
func (e AssetSecretNotFoundError) Error() string {
	return fmt.Sprintf("Secret %v not found in namespace %v", e.Name, e.Namespace)
}

// NewAssetSecretNotFoundError returns a AssetSecretNotFoundError
func NewAssetSecretNotFoundError(name, namespace string) error {
	return &AssetSecretNotFoundError{
		Name:      name,
		Namespace: namespace,
	}
}

// IsAssetSecretNotFoundError returns true if the err is a AssetSecretNotFoundError
func IsAssetSecretNotFoundError(err error) bool {
	_, ok := err.(*AssetSecretNotFoundError)
	return ok
}
