// +build integration

package actions_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestManagedClusterActions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ManagedClusterActions Suite")
}
