// +build integration
package works_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWork(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Works Suite")
}
