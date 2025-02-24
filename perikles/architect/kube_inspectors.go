package architect

import (
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"time"
)

func (p *PeriklesHandler) StartWatching() error {
	clientset, err := kubernetes.NewForConfig(p.Kube.RestConfig())
	if err != nil {
		return err
	}
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)

	// Watch Pods and Deployments
	podInformer := factory.Core().V1().Pods().Informer()
	deployInformer := factory.Apps().V1().Deployments().Informer()
	jobInformer := factory.Batch().V1().Jobs().Informer()

	// Register event handlers
	podInformer.AddEventHandler(p.handlePodEvents())
	deployInformer.AddEventHandler(p.handleDeploymentEvents())
	jobInformer.AddEventHandler(p.handleJobEvents())

	// Start informers
	stopCh := make(chan struct{})
	factory.Start(stopCh)

	<-stopCh // Keep running indefinitely
	return nil
}

func (p *PeriklesHandler) handlePodEvents() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				logging.Error("failed to cast obj to Pod")
				return
			}

			if pod.Namespace != p.Namespace {
				return
			}

			deployment, err := p.podPartOfADeployment(pod)
			if err != nil {
				logging.Debug(fmt.Sprintf("pod not part of a deployment: %s no action needed", pod.Name))
			}

			if deployment != nil {
				if hostsAnnotation, exists := deployment.Spec.Template.Annotations[AnnotationHost]; exists {
					secretName := fmt.Sprintf("%s-tls-certs", hostsAnnotation)
					err = p.ensureSecrets(secretName)
					if err != nil {
						err := p.checkForAnnotations(deployment)
						if err != nil {
							logging.Error(err.Error())
							return
						}
					}

					logging.Debug(fmt.Sprintf("secrets exists for deployment %s trigger by creation of pod: %s", deployment.Name, pod.Name))
				}

				return
			}

			job, err := p.podPartOfAJob(pod)
			if err != nil {
				logging.Debug(fmt.Sprintf("pod not part of a job: %s no action needed", pod.Name))
			}

			if job != nil {
				err := p.checkForElasticAnnotations(nil, job)
				if err != nil {
					logging.Error(err.Error())
					return
				}
			}
		},
	}
}

func (p *PeriklesHandler) handleDeploymentEvents() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deploy, ok := obj.(*appsv1.Deployment)
			if !ok {
				logging.Error("failed to cast obj to Deployment")
				return
			}

			if deploy.Namespace != p.Namespace {
				return
			}

			logging.System(fmt.Sprintf("deploy created: name=%s, namespace=%s", deploy.Name, deploy.Namespace))
			err := p.checkForElasticAnnotations(deploy, nil)
			if err != nil {
				logging.Error(err.Error())
			}
		},
		DeleteFunc: func(obj interface{}) {
			deploy, ok := obj.(*appsv1.Deployment)
			if !ok {
				logging.Error("failed to cast obj to Deployment")
				return
			}

			if deploy.Namespace != p.Namespace {
				return
			}

			logging.System(fmt.Sprintf("deploy deleted: name=%s, namespace=%s", deploy.Name, deploy.Namespace))
			if err := p.cleanUpNetWorkPolicies(deploy.Name); err != nil {
				logging.Error(fmt.Sprintf("Failed to clean up network policies for %s: %v", deploy.Name, err))
			}
			// Clean up mapping
			if err := p.cleanUpMapping(deploy.Name); err != nil {
				logging.Error(fmt.Sprintf("failed to clean up mapping for service %s: %v", deploy.Name, err))
			}
		},
	}
}

func (p *PeriklesHandler) handleJobEvents() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			job, ok := obj.(*batchv1.Job)
			if !ok {
				logging.Error("failed to cast obj to Job")
				return
			}

			if job.Namespace != p.Namespace {
				return
			}

			logging.System(fmt.Sprintf("job created: name=%s, namespace=%s", job.Name, job.Namespace))
			err := p.checkForElasticAnnotations(nil, job)
			if err != nil {
				logging.Error(err.Error())
			}
		},
		DeleteFunc: func(obj interface{}) {
			job, ok := obj.(*batchv1.Job)
			if !ok {
				logging.Error("failed to cast obj to Deployment")
				return
			}

			if job.Namespace != p.Namespace {
				return
			}

			logging.System(fmt.Sprintf("job deleted: name=%s, namespace=%s", job.Name, job.Namespace))
			err := p.cleanUpNetWorkPolicies(job.Name)
			if err != nil {
				logging.Error(err.Error())
			}
		},
	}
}
