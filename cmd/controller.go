package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	set "github.com/deckarep/golang-set"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	logger     *logrus.Entry
	kubeClient kubernetes.Interface
	queue      workqueue.RateLimitingInterface
	informer   cache.SharedIndexInformer
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.logger.Info("Starting the sidecar controller")
	go c.informer.Run(stopCh)

	// Wait for cache synchronization before starting the workqueue
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	c.logger.Info("Sidecar controller is ready")

	wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
		// Continue loop and process items
	}
}

func (c *Controller) processNextItem() bool {
	// Get next item's key
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	// At end of this function, key is processed
	defer c.queue.Done(key)

	err := c.processItem(key.(string))

	if err == nil {
		c.queue.Forget(key)
	} else if c.queue.NumRequeues(key) < 3 {
		// Requeue if max retries not reached
		c.logger.Errorf("Error processing %s (will retry): %v", key, err)
		c.queue.AddRateLimited(key)
	} else {
		// Max retries exceeded. Give up
		c.logger.Errorf("Error processing %s (giving up): %v", key, err)
		c.queue.Forget(key)
		utilruntime.HandleError(err)
	}

	return true // Return true to keep process
}

func (c *Controller) processItem(key string) error {
	obj, exists, err := c.informer.GetIndexer().GetByKey(key)

	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"key": key,
		}).Errorf("Error fetching object from store: %v", err)
		return fmt.Errorf("Error fetching object with key %s from store: %v", key, err)
	}

	if !exists {
		c.logger.WithFields(logrus.Fields{
			"key": key,
		}).Debug("Does not exist. Pass")
		return nil
	}

	pod := obj.(*corev1.Pod)
	sidecarStr := pod.Annotations["limgit/sidecars"]
	if sidecarStr == "" {
		c.logger.WithFields(logrus.Fields{
			"key":    key,
			"status": pod.Status.Phase,
		}).Trace("No `limgit/sidecars` annotation. Pass")
		return nil
	}

	allContainers := set.NewSet()
	runningContainers := set.NewSet()
	completedContainers := set.NewSet()
	sidecars := set.NewSet()

	for _, s := range strings.Split(sidecarStr, ",") {
		sidecars.Add(s)
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		allContainers.Add(containerStatus.Name)
		if containerStatus.Ready {
			runningContainers.Add(containerStatus.Name)
		} else {
			terminated := containerStatus.State.Terminated
			if terminated != nil && (terminated.Reason == "Completed" || terminated.Reason == "Error") {
				completedContainers.Add(containerStatus.Name)
			}
		}
	}

	logFields := logrus.Fields{
		"key":        key,
		"status":     pod.Status.Phase,
		"cTotal":     allContainers.Cardinality(), // c stands for container
		"cRunning":   runningContainers.Cardinality(),
		"cCompleted": completedContainers.Cardinality(),
		"cSidecars":  sidecars.Cardinality(),
	}
	c.logger.WithFields(logFields).Debug("`limgit/sidecars` annotation found")

	if runningContainers.Union(completedContainers).Equal(allContainers) {
		// All the containers are running or completed. If the running containers are all sidecars, shutdown
		if runningContainers.Equal(sidecars) {
			c.logger.WithFields(logFields).Debug("Only sidecar containers are remaining. Shutdown them")
			for _, container := range sidecars.ToSlice() {
				stderr, err := execCommand(c.kubeClient, pod, container.(string), "kill -s TERM 1")
				if err != nil {
					c.logger.WithFields(logFields).Errorf("stderr invoking kill command: %s", err.Error())
				}
				if len(stderr) > 0 {
					c.logger.WithFields(logFields).Errorf("stderr invoking kill command: %s", string(stderr))
				}
			}
		}
	}
	return nil
}

func execCommand(kubeClient kubernetes.Interface, pod *corev1.Pod, container string, command string) ([]byte, error) {
	req := kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"/bin/sh", "-c", command},
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(getKubeConfig(), "POST", req.URL())
	if err != nil {
		return nil, err
	}

	var stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return nil, err
	}

	return stderr.Bytes(), nil
}
