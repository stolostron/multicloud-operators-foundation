package getter

import (
	"context"
	"errors"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	fakeaddon "open-cluster-management.io/api/client/addon/clientset/versioned/fake"
	"strconv"
	"testing"
)

func newAddon(namespace, name string) *addonv1alpha1.ManagedClusterAddOn {
	return &addonv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
}
func newLogSASecret(namespace string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: helpers.LogManagedServiceAccountName, Namespace: namespace},
	}
}

func TestProxyServiceAvailable(t *testing.T) {
	cases := []struct {
		name        string
		clusterName string
		addons      []runtime.Object
		secret      []runtime.Object
		expectedRst bool
		expectedErr error
	}{
		{
			name:        "no addons",
			clusterName: "cluster1",
			expectedRst: false,
			expectedErr: nil,
		},
		{
			name:        "no msa addon",
			clusterName: "cluster1",
			addons:      []runtime.Object{newAddon("cluster1", helpers.ClusterProxyAddonName)},
			expectedRst: false,
			expectedErr: nil,
		},
		{
			name:        "no cluster proxy addon",
			clusterName: "cluster1",
			addons:      []runtime.Object{newAddon("cluster1", helpers.MsaAddonName)},
			expectedRst: false,
			expectedErr: nil,
		},
		{
			name:        "have managed serviceaccount and cluster proxy addon",
			clusterName: "cluster1",
			addons: []runtime.Object{newAddon("cluster1", helpers.MsaAddonName),
				newAddon("cluster1", helpers.ClusterProxyAddonName)},
			expectedRst: true,
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakeKubeClient := fake.NewSimpleClientset(c.secret...)
			fakeAddonClient := fakeaddon.NewSimpleClientset(c.addons...)
			logProxyGetter := NewLogProxyGetter(fakeAddonClient, fakeKubeClient, "", "")
			rst, err := logProxyGetter.ProxyServiceAvailable(context.TODO(), c.clusterName)
			if !errors.Is(err, c.expectedErr) {
				t.Errorf("unexpected err.%v", err)
			}
			if rst != c.expectedRst {
				t.Errorf("unexpected rst.%v", rst)
			}
		})
	}
}

func TestNewHandler(t *testing.T) {
	fakeKubeClient := fake.NewSimpleClientset(newLogSASecret("cluster1"))
	fakeAddonClient := fakeaddon.NewSimpleClientset(newAddon("cluster1", helpers.MsaAddonName),
		newAddon("cluster1", helpers.ClusterProxyAddonName))
	logProxyGetter := NewLogProxyGetter(fakeAddonClient, fakeKubeClient,
		"cluster-proxy-addon-user.open-cluster-management.svc", "./")

	_, _ = logProxyGetter.NewHandler(context.TODO(), "cluster1", "default", "test", "test")

}

func TestHandler(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: v1.PodSpec{
			Volumes:        nil,
			InitContainers: nil,
			Containers: []v1.Container{
				{
					Name: "test",
				},
			},
		},
		Status: v1.PodStatus{},
	}
	fakeKubeClient := fake.NewSimpleClientset(pod)
	handler := Handler{
		logClient:     fakeKubeClient,
		podNamespace:  "default",
		podName:       "test",
		containerName: "test",
	}
	r, _ := http.NewRequest(http.MethodOptions,
		"clusterstatuses/mcroshfit/log/default/test/test?tailLines=1000&follow=true&previous=true&timestamps=true&sinceSeconds=100", nil)

	handler.ServeHTTP(&fakeWriter{}, r)
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
	tempHead["statusCode"] = []string{strconv.Itoa(status)}
	fw.header = tempHead
}

func (fw *fakeWriter) Write(data []byte) (int, error) {
	return len(data), nil
}
