package worksets_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWorksets(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Worksets Suite")
}
