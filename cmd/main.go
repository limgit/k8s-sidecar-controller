// Basic structure inspired by https://engineering.bitnami.com/articles/a-deep-dive-into-kubernetes-controllers.html
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func initApp() {
	initConfig()
	initLogger()
}

func main() {
	initApp()
	kubeClient := initKubeClient()
	logger := getLogger()

	logger.Info("Initializing sidecar controller")
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Create shared informer which gets informed by K8S events
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().Pods(metav1.NamespaceAll).Watch(context.TODO(), options)
			},
		},
		&corev1.Pod{},
		0, // Do not resync
		cache.Indexers{},
	)

	// Passing pod events to the workqueue
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj) // pod_namespace/pod_name
			if err == nil {
				queue.Add(key)
			}
		},
	})

	controller := Controller{
		logger:     logger,
		kubeClient: kubeClient,
		informer:   informer,
		queue:      queue,
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	go controller.Run(stopCh)

	endCh := make(chan os.Signal, 1)
	signal.Notify(endCh, syscall.SIGTERM)
	signal.Notify(endCh, syscall.SIGINT)
	<-endCh

	logger.Info("Shutting down requested. Cleaning up.")
}
