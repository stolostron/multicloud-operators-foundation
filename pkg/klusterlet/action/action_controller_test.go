package controllers

import (
	"testing"
	"time"

	tlog "github.com/go-logr/logr/testing"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"
)

var c client.Client

const (
	actionName      = "name"
	actionNamespace = "default"
)

func TestControllerReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	kubework := NewKubeWorkSpec()
	action := NewAction(actionName, actionNamespace, actionv1beta1.DeleteActionType, kubework)
	ar := NewActionReconciler(c, tlog.NullLogger{}, mgr.GetScheme(), nil, rest.NewFakeKubeControl(), false)

	ar.SetupWithManager(mgr)

	SetupTestReconcile(ar)
	ar.SetupWithManager(mgr)

	cancel, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		cancel()
		mgrStopped.Wait()
	}()

	// Create the object and expect the Reconcile
	err = c.Create(context.TODO(), action)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), action)

	time.Sleep(time.Second * 1)
}
