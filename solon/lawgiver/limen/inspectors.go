package limen

import (
	"fmt"
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func (c *Client) StartWatching() error {
	if c == nil || c.kubeClient == nil {
		return fmt.Errorf("limen client is not initialized with a kube client")
	}

	clientset, err := kubernetes.NewForConfig(c.kubeClient.RestConfig())
	if err != nil {
		return err
	}
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)

	// Watch Pods and Deployments
	podInformer := factory.Core().V1().Pods().Informer()

	// Register event handlers
	podInformer.AddEventHandler(c.handlePodEvents())

	// Start informers
	stopCh := make(chan struct{})
	factory.Start(stopCh)

	<-stopCh // Keep running indefinitely
	return nil
}

func (c *Client) handlePodEvents() cache.ResourceEventHandlerFuncs {
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
				err := c.deleteOrphans(pod)
				if err != nil {
					logging.Error(err.Error())
				}
			}
		},
	}
}
