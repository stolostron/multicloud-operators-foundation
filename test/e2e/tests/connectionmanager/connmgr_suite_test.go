// +build integration

package connmgr_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConnectionManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConnectionManager Suite")
}
