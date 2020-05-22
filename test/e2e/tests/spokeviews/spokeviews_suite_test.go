// +build integration

package spokeviews_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestClusterActions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SpokeViews Suite")
}
