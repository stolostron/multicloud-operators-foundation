// +build integration

package clusterinfos_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestManagedClusterInfos(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ManagedClusterInfo Suite")
}
