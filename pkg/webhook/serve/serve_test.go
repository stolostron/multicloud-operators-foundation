package serve

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	v1 "k8s.io/api/admission/v1"
)

var statusCode = "statusCode"

func fakeadmit(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	status := &v1.AdmissionResponse{
		Allowed: true,
	}
	return status
}
func TestServe(t *testing.T) {
	cases := []struct {
		name           string
		request        *http.Request
		responseWriter *fakeWriter
	}{
		{
			name: "nil body in request",
			request: func() *http.Request {
				r, _ := http.NewRequest(http.MethodOptions, "url", nil)
				return r
			}(),

			responseWriter: &fakeWriter{},
		},
		{
			name: "error header format in request",
			request: func() *http.Request {
				r, _ := http.NewRequest(http.MethodOptions, "url", strings.NewReader("{\"foo\":\"bar\"}"))
				return r
			}(),

			responseWriter: &fakeWriter{},
		},
		{
			name: "requestedAdmissionReview is not right",
			request: func() *http.Request {
				ctx := context.TODO()
				r, _ := http.NewRequestWithContext(ctx, http.MethodHead, "url", strings.NewReader("{\"foo\":\"bar\"}"))
				r.Header.Add("Content-Type", "application/json")
				return r
			}(),
			responseWriter: &fakeWriter{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			Serve(c.responseWriter, c.request, fakeadmit)
			if len(c.responseWriter.Header()[statusCode]) == 0 {
				t.Errorf("response error:%v", c.responseWriter.Header())
			}
		})
	}
}

type fakeWriter struct {
	header http.Header
}

func (fw *fakeWriter) Header() http.Header {
	if fw.header == nil {
		fw.header = http.Header{}
	}
	return fw.header
}

func (fw *fakeWriter) WriteHeader(status int) {
	tempHead := make(map[string][]string)
	tempHead[statusCode] = []string{strconv.Itoa(status)}
	fw.header = tempHead
}

func (fw *fakeWriter) Write(data []byte) (int, error) {
	return len(data), nil
}
