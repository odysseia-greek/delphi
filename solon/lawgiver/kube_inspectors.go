package lawgiver

import (
	"github.com/odysseia-greek/agora/plato/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"
)

func (s *SolonHandler) StartWatching() error {
	clientset, err := kubernetes.NewForConfig(s.Kube.RestConfig())
	if err != nil {
		return err
	}
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)

	// Watch Pods and Deployments
	podInformer := factory.Core().V1().Pods().Informer()

	// Register event handlers
	podInformer.AddEventHandler(s.handlePodEvents())

	// Start informers
	stopCh := make(chan struct{})
	factory.Start(stopCh)

	<-stopCh // Keep running indefinitely
	return nil
}

func (s *SolonHandler) handlePodEvents() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				logging.Error("failed to cast obj to Pod")
				return
			}

			if pod.Namespace != s.Namespace {
				return
			}

			err := s.deleteOrphans(pod)
			if err != nil {
				logging.Error(err.Error())
			}
		},
	}
}
