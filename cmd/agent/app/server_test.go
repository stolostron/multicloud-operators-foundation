package app

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	addonutils "open-cluster-management.io/addon-framework/pkg/utils"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var tmpFileName string

var _ = BeforeSuite(func() {
	tmpFile, err := os.CreateTemp("/tmp", "test")
	Expect(err).To(BeNil())

	tmpFileName = tmpFile.Name()

	cc, err := addonutils.NewConfigChecker("test", tmpFileName)
	Expect(err).To(BeNil())

	go ServeHealthProbes(context.Background().Done(), ":8000", cc.Check)
})

var _ = Describe("Config changed", func() {
	BeforeEach(func() {
		// pre check
		resp, err := http.Get("http://localhost:8000/healthz")
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		// change the config
		err = os.WriteFile(tmpFileName, []byte("changed content"), os.ModeAppend)
		Expect(err).To(BeNil())
	})
	It("should return false", func() {
		// check
		resp, err := http.Get("http://localhost:8000/healthz")
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

		time.Sleep(3 * time.Second)
		// check again
		resp, err = http.Get("http://localhost:8000/healthz")
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
	})
})
