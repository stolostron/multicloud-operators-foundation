// +build integration

package cluster_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestManagedcluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ManagedCluster Suite")
}
