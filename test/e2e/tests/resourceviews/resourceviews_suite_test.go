// +build integration

package resourceviews_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestResourceviews(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resourceviews Suite")
}
