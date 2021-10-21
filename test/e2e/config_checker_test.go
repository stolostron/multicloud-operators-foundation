package e2e

import (
	"context"
	"fmt"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Change the secret of hub-kubeconfig", func() {
	namespace := "open-cluster-management-agent-addon"
	var podName string
	var containerRestartCount int

	BeforeEach(func() {
		// get work-manager's pod
		pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: "app=work-manager",
		})
		Expect(err).To(BeNil())
		Expect(len(pods.Items)).ToNot(Equal(0))
		By("pods number: " + strconv.Itoa(len(pods.Items)))

		// get podName
		podName = pods.Items[0].Name
		containerRestartCount = int(pods.Items[0].Status.ContainerStatuses[0].RestartCount)

		By("pod name: " + podName)
		By("container Restart Count: " + strconv.Itoa(containerRestartCount))

		// make sure container are runing before test
		Eventually(func() bool {
			pod, err := kubeClient.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			return pod.Status.ContainerStatuses[0].State.Running.StartedAt.IsZero()
		}, eventuallyTimeout, eventuallyInterval).Should(Equal(false))
	})

	It("shoud restart the container of work manager", func() {
		// print currnet pod status
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		By(pod.String())

		// change the secret of work-manager
		secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), "work-manager-hub-kubeconfig", metav1.GetOptions{})
		Expect(err).To(BeNil())

		content := string(secret.Data["kubeconfig"])
		By(fmt.Sprintf("secret content:%s\n", content))
		secret.Data["kubeconfig"] = []byte(content + " # add one line to trigger containter restart")

		_, err = kubeClient.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
		Expect(err).To(BeNil())

		Eventually(func() bool {
			// get pod again and check the containerRestartCount
			pod, err := kubeClient.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			By("latest container restart count: " + strconv.Itoa(int(pod.Status.ContainerStatuses[0].RestartCount)))
			return containerRestartCount < int(pod.Status.ContainerStatuses[0].RestartCount)
		}, eventuallyTimeout, eventuallyInterval).Should(Equal(true))
	})
})
