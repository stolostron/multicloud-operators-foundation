// +build integration

package clusteractions_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestClusterActions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ClusterActions Suite")
}
