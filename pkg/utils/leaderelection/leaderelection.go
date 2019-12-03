// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

// usage example:
//func Run(s *options.RunOptions, stopCh <-chan struct{}) error {
//	genericConfig, err := s.Generic.BuildConfig()
//	if err != nil {
//		klog.Fatalf("Error build config: %s", err.Error())
//	}
//
//	run := func(stopCh <-chan struct{})error{
//		err = RunOperator(s, genericConfig, stopCh)
//		if err != nil {
//			klog.Fatalf("Error run operator: %s", err.Error())
//		}
//		return nil
//	}
//
//	if err := leaderelection.Run(s.LeaderElect,genericConfig.Kubeclient,"kube-system","test-test",stopCh,run);err != nil {
//		klog.Fatalf("Error leaderelection run: %s", err.Error())
//	}
//}

package leaderelection

import (
	"context"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

type runFunc func(stopCh <-chan struct{}) error

// Run runs runFunc according value of leaderEnabled.
// Run uses kubeClient to create a configmap named componentName in the namespace to select leader.
// componentName is the name of configmap that has leader info, must be unique in the namespace.
func Run(leaderEnabled bool, kubeClient kubernetes.Interface, namespace, componentName string, stopCh <-chan struct{}, runFunc runFunc) error {
	if leaderEnabled {
		run := func(ctx context.Context) {
			if err := runFunc(stopCh); err != nil {
				klog.Fatalf("Error run func: %s", err.Error())
			}
			klog.Info("I am leading")
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-stopCh
			cancel()
		}()

		if err := createAndRun(ctx, componentName, namespace, kubeClient, run); err != nil {
			klog.Fatalf("Error create component and run leader election %s", err.Error())
		}
	} else if err := runFunc(stopCh); err != nil {
		klog.Fatalf("Error run func: %s", err.Error())
	}
	<-stopCh
	return nil
}

// createAndRun creates the resource of leader election and run leader election.
func createAndRun(ctx context.Context, componentName string, namespace string,
	kubeClient kubernetes.Interface, run func(context.Context)) error {
	lock, err := createResourceLock(kubeClient, componentName, namespace)
	if err != nil {
		klog.Errorf("Error create resource lock: %s", err.Error())
		return err
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				klog.Errorf("Error component leader election lost")
			},
		},
		Name: componentName,
	})

	return nil
}

func createResourceLock(kubeClient kubernetes.Interface, componentName string,
	namespace string) (resourcelock.Interface, error) {
	id := os.Getenv("HOSTNAME")
	if id == "" {
		id = componentName + "-" + string(uuid.NewUUID())
	}

	eventRecorder, err := createRecorder(kubeClient, componentName)
	if err != nil {
		klog.Errorf("Error create recorder: %v", err)
		return nil, err
	}

	lock, err := resourcelock.New(resourcelock.ConfigMapsResourceLock,
		namespace,
		componentName+"-leader-election",
		kubeClient.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: eventRecorder,
		})
	if err != nil {
		klog.Errorf("Error create resource lock: %v", err)
		return nil, err
	}

	return lock, nil
}

func createRecorder(kubeClient kubernetes.Interface, componentName string) (record.EventRecorder, error) {
	eventsScheme := runtime.NewScheme()
	if err := v1.AddToScheme(eventsScheme); err != nil {
		klog.Errorf("Error add to scheme : %v", err)
		return nil, err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})

	return eventBroadcaster.NewRecorder(eventsScheme, v1.EventSource{Component: componentName}), nil
}
