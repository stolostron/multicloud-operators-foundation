package getter

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

// TODO: Add more testing cases
func TestLogConnectionInfoGetter(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(clusterInfoGVR.GroupVersion())
	fakeDynamicClient := fake.NewSimpleDynamicClient(scheme)

	config := ClientConfig{
		DynamicClient: fakeDynamicClient,
	}

	getter, err := NewLogConnectionInfoGetter(config)
	if err != nil {
		t.Errorf("Failed to run NewLogConnectionInfoGetter: %v", err)
	}

	getter.GetConnectionInfo(context.TODO(), "")
}
