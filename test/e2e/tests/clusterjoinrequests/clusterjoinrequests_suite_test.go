// +build integration

package clusterjoinrequests_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestClusterjoinrequests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Clusterjoinrequests Suite")
}
