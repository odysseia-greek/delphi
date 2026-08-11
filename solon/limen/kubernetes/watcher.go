package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func (c *Client) StartWatching() error {
	if c == nil || c.Interface == nil {
		return fmt.Errorf("limen client is not initialized with a kube client")
	}

	factory := informers.NewSharedInformerFactory(c, 30*time.Second)

	// Watch Pods and Deployments
	podInformer := factory.Core().V1().Pods().Informer()

	// Register event handlers
	podInformer.AddEventHandler(c.handlePodWatchEvents())

	// Start informers
	stopCh := make(chan struct{})
	factory.Start(stopCh)

	<-stopCh // Keep running indefinitely
	return nil
}

func (c *Client) handlePodWatchEvents() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				logging.Error("failed to cast obj to Pod")
				return
			}

			var inManagedNameSpace bool
			if pod.Namespace == c.namespaces.SolonNamespace {
				inManagedNameSpace = true
			}

			for _, ns := range c.namespaces.WatchedNamespaces {
				if pod.Namespace == ns {
					inManagedNameSpace = true
					break
				}
			}

			if inManagedNameSpace {
				if c.cleaner == nil {
					logging.Error("limen kubernetes client has no cleanup client")
					return
				}
				username := normalizeUsername(pod.Name)
				err := c.cleaner.DeleteOrphan(context.Background(), username, pod.Name)
				if err != nil {
					logging.Error(err.Error())
				}
			}
		},
	}
}

func normalizeUsername(podName string) string {
	splitPodName := strings.Split(podName, "-")
	if len(splitPodName) > 1 {
		return splitPodName[0] + splitPodName[len(splitPodName)-1]
	}
	return splitPodName[0]
}
