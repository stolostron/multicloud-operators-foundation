// +build integration

package views_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestManagedClusterViews(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ManagedClusterView Suite")
}
