package architect

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// StartWatching starts shared informers for pods, deployments, jobs, and namespaces.
// It blocks forever (until stopCh is closed).
func (p *PeriklesHandler) StartWatching() error {
	clientset, err := kubernetes.NewForConfig(p.Kube.RestConfig())
	if err != nil {
		return err
	}

	// Resync is fine; real-time events still come via watch.
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)

	podInformer := factory.Core().V1().Pods().Informer()
	deployInformer := factory.Apps().V1().Deployments().Informer()
	jobInformer := factory.Batch().V1().Jobs().Informer()
	nsInformer := factory.Core().V1().Namespaces().Informer()

	podInformer.AddEventHandler(p.handlePodEvents())
	deployInformer.AddEventHandler(p.handleDeploymentEvents())
	jobInformer.AddEventHandler(p.handleJobEvents())
	nsInformer.AddEventHandler(p.handleNamespaceEvents())

	stopCh := make(chan struct{})

	logging.System("Starting informers (pods, deployments, jobs, namespaces)...")
	factory.Start(stopCh)

	logging.System("Waiting for informer caches to sync...")
	if ok := cache.WaitForCacheSync(
		stopCh,
		podInformer.HasSynced,
		deployInformer.HasSynced,
		jobInformer.HasSynced,
		nsInformer.HasSynced,
	); !ok {
		return fmt.Errorf("failed waiting for caches to sync")
	}
	logging.System("Informer caches synced. Watching for events.")

	<-stopCh
	return nil
}

func (p *PeriklesHandler) isManagedNamespace(ns string) bool {
	if ns == p.Namespace {
		return true
	}
	for _, watched := range p.WatchedNamespaces {
		if ns == watched {
			return true
		}
	}
	return false
}

func (p *PeriklesHandler) handlePodEvents() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				logging.Error("failed to cast obj to Pod")
				return
			}

			var inManagedNameSpace bool
			if pod.Namespace == p.Namespace {
				inManagedNameSpace = true
			}

			if !inManagedNameSpace {
				for _, ns := range p.WatchedNamespaces {
					if pod.Namespace == ns {
						inManagedNameSpace = true
						break
					}
				}
			}

			if !inManagedNameSpace {
				return
			}

			deployment, err := p.podPartOfADeployment(pod)
			if err != nil {
				logging.Debug(fmt.Sprintf("pod not part of a deployment: %s no action needed", pod.Name))
			}

			if deployment != nil {
				if hostsAnnotation, exists := deployment.Spec.Template.Annotations[AnnotationHost]; exists {
					secretName := fmt.Sprintf("%s-tls-certs", hostsAnnotation)
					err = p.ensureSecrets(secretName, deployment.Namespace)
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
		DeleteFunc: func(obj interface{}) {
			pod, ok := podFromDeleteEvent(obj)
			if !ok {
				logging.Error("failed to cast deleted object to Pod")
				return
			}
			if !p.isManagedNamespace(pod.Namespace) {
				return
			}

			for _, owner := range pod.OwnerReferences {
				if owner.Kind != "Job" {
					continue
				}

				active, err := p.jobHasActivePods(owner.Name, pod.Namespace)
				if err != nil {
					logging.Error(fmt.Sprintf("failed to check active pods for job %s: %v", owner.Name, err))
					return
				}
				if active {
					logging.Debug(fmt.Sprintf("job %s still has an active pod; keeping its network policies", owner.Name))
					return
				}

				logging.System(fmt.Sprintf("job pod deleted: name=%s, job=%s, namespace=%s", pod.Name, owner.Name, pod.Namespace))
				if err := p.cleanUpNetWorkPolicies(owner.Name, pod.Namespace); err != nil {
					logging.Error(fmt.Sprintf("failed to clean up network policies for job %s: %v", owner.Name, err))
				}
				return
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

			var inManagedNameSpace bool
			if deploy.Namespace == p.Namespace {
				inManagedNameSpace = true
			}

			if !inManagedNameSpace {
				for _, ns := range p.WatchedNamespaces {
					if deploy.Namespace == ns {
						inManagedNameSpace = true
						break
					}
				}
			}

			if !inManagedNameSpace {
				return
			}

			logging.System(fmt.Sprintf("deploy created: name=%s, namespace=%s", deploy.Name, deploy.Namespace))
			p.recordEvent("deployment.created", "Deployment discovered", deploy.Namespace, deploy.Name, nil)
			if err := p.checkForAnnotations(deploy); err != nil {
				logging.Error(err.Error())
			}
			err := p.checkForElasticAnnotations(deploy, nil)
			if err != nil {
				logging.Error(err.Error())
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeploy, oldOK := oldObj.(*appsv1.Deployment)
			newDeploy, newOK := newObj.(*appsv1.Deployment)
			if !oldOK || !newOK {
				logging.Error("failed to cast updated object to Deployment")
				return
			}
			if !p.isManagedNamespace(newDeploy.Namespace) {
				return
			}
			if reflect.DeepEqual(oldDeploy.Spec.Template.Annotations, newDeploy.Spec.Template.Annotations) {
				return
			}

			p.recordEvent("deployment.updated", "Deployment annotations changed", newDeploy.Namespace, newDeploy.Name, nil)
			if err := p.checkForAnnotations(newDeploy); err != nil {
				logging.Error(err.Error())
			}
			if err := p.checkForElasticAnnotations(newDeploy, nil); err != nil {
				logging.Error(err.Error())
			}
		},
		DeleteFunc: func(obj interface{}) {
			deploy, ok := obj.(*appsv1.Deployment)
			if !ok {
				logging.Error("failed to cast obj to Deployment")
				return
			}

			var inManagedNameSpace bool
			if deploy.Namespace == p.Namespace {
				inManagedNameSpace = true
			}

			if !inManagedNameSpace {
				for _, ns := range p.WatchedNamespaces {
					if deploy.Namespace == ns {
						inManagedNameSpace = true
						break
					}
				}
			}

			if !inManagedNameSpace {
				return
			}

			logging.System(fmt.Sprintf("deploy deleted: name=%s, namespace=%s", deploy.Name, deploy.Namespace))
			p.recordEvent("deployment.deleted", "Deployment deleted", deploy.Namespace, deploy.Name, nil)
			if err := p.cleanUpNetWorkPolicies(deploy.Name, deploy.Namespace); err != nil {
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

			var inManagedNameSpace bool
			if job.Namespace == p.Namespace {
				inManagedNameSpace = true
			}

			if !inManagedNameSpace {
				for _, ns := range p.WatchedNamespaces {
					if job.Namespace == ns {
						inManagedNameSpace = true
						break
					}
				}
			}

			if !inManagedNameSpace {
				return
			}

			logging.System(fmt.Sprintf("job created: name=%s, namespace=%s", job.Name, job.Namespace))
			p.recordEvent("job.created", "Job discovered", job.Namespace, job.Name, nil)
			err := p.checkForElasticAnnotations(nil, job)
			if err != nil {
				logging.Error(err.Error())
			}
		},
		DeleteFunc: func(obj interface{}) {
			job, ok := jobFromDeleteEvent(obj)
			if !ok {
				logging.Error("failed to cast deleted object to Job")
				return
			}

			var inManagedNameSpace bool
			if job.Namespace == p.Namespace {
				inManagedNameSpace = true
			}

			if !inManagedNameSpace {
				for _, ns := range p.WatchedNamespaces {
					if job.Namespace == ns {
						inManagedNameSpace = true
						break
					}
				}
			}

			if !inManagedNameSpace {
				return
			}

			logging.System(fmt.Sprintf("job deleted: name=%s, namespace=%s", job.Name, job.Namespace))
			p.recordEvent("job.deleted", "Job deleted", job.Namespace, job.Name, nil)
			err := p.cleanUpNetWorkPolicies(job.Name, job.Namespace)
			if err != nil {
				logging.Error(err.Error())
			}
		},
	}
}

func podFromDeleteEvent(obj interface{}) (*v1.Pod, bool) {
	if pod, ok := obj.(*v1.Pod); ok {
		return pod, true
	}

	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if !ok {
		return nil, false
	}
	pod, ok := tombstone.Obj.(*v1.Pod)
	return pod, ok
}

func jobFromDeleteEvent(obj interface{}) (*batchv1.Job, bool) {
	if job, ok := obj.(*batchv1.Job); ok {
		return job, true
	}

	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if !ok {
		return nil, false
	}
	job, ok := tombstone.Obj.(*batchv1.Job)
	return job, ok
}

func (p *PeriklesHandler) handleNamespaceEvents() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ns, ok := obj.(*corev1.Namespace)
			if !ok {
				logging.Error("failed to cast obj to Namespace")
				return
			}
			p.onNamespace(ns, "created")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			ns, ok := newObj.(*corev1.Namespace)
			if !ok {
				logging.Error("failed to cast obj to Namespace")
				return
			}
			p.onNamespace(ns, "updated")
		},
	}
}

func (p *PeriklesHandler) ensureNamespaceBootstrapVaultAndSolon(targetNamespace string) error {
	if targetNamespace == BootstrapSourceNamespace {
		return nil
	}

	logging.System(fmt.Sprintf("namespace %s: replicating Vault TLS secret (%s/%s -> %s/%s)",
		targetNamespace, BootstrapSourceNamespace, VaultTLSSourceSecret, targetNamespace, VaultTLSSourceSecret))

	if err := p.copySecretAllKeys(BootstrapSourceNamespace, targetNamespace, VaultTLSSourceSecret); err != nil {
		return err
	}

	logging.System(fmt.Sprintf("namespace %s: replicating Solon TLS secret (%s/%s -> %s/%s)",
		targetNamespace, BootstrapSourceNamespace, SolonTLSSourceSecret, targetNamespace, SolonTLSSourceSecret))

	if err := p.copySecretAllKeys(BootstrapSourceNamespace, targetNamespace, SolonTLSSourceSecret); err != nil {
		return err
	}

	return nil
}

func (p *PeriklesHandler) onNamespace(ns *corev1.Namespace, verb string) {
	annotations := ns.Annotations
	if annotations == nil {
		return
	}

	targetNS := ns.Name

	// 1) Replication (allowed even if not in managed namespaces)
	if replVal := strings.TrimSpace(annotations[AnnotationReplicateFrom]); replVal != "" {
		refs := parseReplicateFrom(replVal)
		if len(refs) > 0 {
			logging.System(fmt.Sprintf("namespace %s: %s (replicate-from=%q)", targetNS, verb, replVal))
		}

		for _, ref := range refs {
			// Avoid self-copy loops
			if ref.Namespace == targetNS {
				logging.Warn(fmt.Sprintf(
					"namespace %s: skipping replicate-from %s:%s (source and target namespace are the same)",
					targetNS, ref.SecretName, ref.Namespace))
				continue
			}

			logging.System(fmt.Sprintf(
				"namespace %s: replicating secret %s from %s",
				targetNS, ref.SecretName, ref.Namespace))

			// copies the full secret (all keys), as requested
			if err := p.copySecretAllKeys(ref.Namespace, targetNS, ref.SecretName); err != nil {
				logging.Error(fmt.Sprintf(
					"namespace %s: failed replicating secret %s from %s: %v",
					targetNS, ref.SecretName, ref.Namespace, err))
				// keep going; don't fail the entire namespace processing
				continue
			}
		}
	}

	// 2) Bootstrap (only for managed namespaces)
	bootVal := strings.ToLower(strings.TrimSpace(annotations[AnnotationNamespaceBootstrap]))
	if bootVal == "true" {
		if !p.isManagedNamespace(targetNS) {
			logging.Debug(fmt.Sprintf(
				"namespace %s: bootstrap=true but namespace is not managed; ignoring bootstrap",
				targetNS))
			return
		}

		logging.System(fmt.Sprintf("namespace %s: %s (bootstrap=true)", targetNS, verb))

		if err := p.ensureNamespaceBootstrapVaultAndSolon(targetNS); err != nil {
			logging.Error(fmt.Sprintf("namespace %s: bootstrap failed: %v", targetNS, err))
			return
		}

		logging.System(fmt.Sprintf(
			"namespace %s: bootstrap complete (vault+solon TLS secrets replicated)",
			targetNS))
	}
}
