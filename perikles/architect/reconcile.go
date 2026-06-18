package architect

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/perikles/pkg/service_mapping/crd/v1alpha"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var legacyCronJobPolicyName = regexp.MustCompile(`^restrict-elasticsearch-access-.+-[0-9]+$`)

func (p *PeriklesHandler) cleanUpNetWorkPolicies(serviceToRemove, ns string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespacesToCheck := []string{ns, p.Namespace, p.ElasticNs, p.VaultNs}
	namespacesToCheck = append(namespacesToCheck, p.WatchedNamespaces...)
	namespaces := uniqueNamespaces(namespacesToCheck...)
	for _, namespace := range namespaces {
		nwpInWatchedNs, err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list network policies in %s namespace: %w", namespace, err)
		}

		for _, nwp := range nwpInWatchedNs.Items {
			if policyBelongsToWorkload(nwp.Name, serviceToRemove) {
				err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(namespace).Delete(ctx, nwp.Name, metav1.DeleteOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						continue
					}
					return fmt.Errorf("failed to delete network policy %s: %w", nwp.Name, err)
				}
				logging.Debug(fmt.Sprintf("Deleted network policy: %s in ns: %s", nwp.Name, namespace))
				p.recordEvent("policy.deleted", "Network policy deleted with its workload", namespace, nwp.Name, map[string]string{
					"workload": serviceToRemove,
				})
			}
		}
	}

	return nil
}

func policyBelongsToWorkload(policyName, workloadName string) bool {
	if policyName == fmt.Sprintf("restrict-elasticsearch-access-%s", workloadName) {
		return true
	}

	return strings.HasPrefix(policyName, fmt.Sprintf("allow-%s-access-", workloadName))
}

func uniqueNamespaces(namespaces ...string) []string {
	seen := make(map[string]struct{}, len(namespaces))
	unique := make([]string, 0, len(namespaces))
	for _, namespace := range namespaces {
		namespace = strings.TrimSpace(namespace)
		if namespace == "" {
			continue
		}
		if _, exists := seen[namespace]; exists {
			continue
		}
		seen[namespace] = struct{}{}
		unique = append(unique, namespace)
	}
	return unique
}

func (p *PeriklesHandler) LoopForStaleNetworkPolicies() {
	interval := p.ReconcileTimer
	if interval <= 0 {
		interval = time.Hour
	}

	if err := p.cleanUpStaleJobNetworkPolicies(time.Now().UTC(), staleJobPolicyAge); err != nil {
		logging.Error(err.Error())
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		if err := p.cleanUpStaleJobNetworkPolicies(time.Now().UTC(), staleJobPolicyAge); err != nil {
			logging.Error(err.Error())
		}
	}
}

func (p *PeriklesHandler) cleanUpStaleJobNetworkPolicies(now time.Time, maxAge time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	policies, err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(p.ElasticNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list elastic network policies in namespace %s: %w", p.ElasticNs, err)
	}

	for _, policy := range policies.Items {
		if !isStaleJobNetworkPolicy(&policy, now, maxAge) {
			continue
		}

		if err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(p.ElasticNs).Delete(ctx, policy.Name, metav1.DeleteOptions{}); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("failed to delete stale network policy %s: %w", policy.Name, err)
		}
		logging.System(fmt.Sprintf("Deleted stale job network policy: %s in ns: %s", policy.Name, p.ElasticNs))
		p.recordEvent("policy.stale_deleted", "Stale job network policy deleted", p.ElasticNs, policy.Name, map[string]string{
			"age": now.Sub(policy.CreationTimestamp.Time).Round(time.Minute).String(),
		})
	}

	return nil
}

func isStaleJobNetworkPolicy(policy *ciliumv2.CiliumNetworkPolicy, now time.Time, maxAge time.Duration) bool {
	if !strings.HasPrefix(policy.Name, "restrict-elasticsearch-access-") {
		return false
	}

	created := policy.CreationTimestamp.Time
	if created.IsZero() {
		updated, err := time.Parse(timeFormat, policy.Annotations[AnnotationUpdate])
		if err != nil {
			return false
		}
		created = updated
	}

	if now.Sub(created) <= maxAge {
		return false
	}

	switch strings.ToLower(policy.Annotations[AnnotationSourceKind]) {
	case "job":
		return true
	case "":
		return legacyCronJobPolicyName.MatchString(policy.Name)
	default:
		return false
	}
}

func (p *PeriklesHandler) jobHasActivePods(jobName, namespace string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pods, err := p.Kube.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("batch.kubernetes.io/job-name=%s", jobName),
	})
	if err != nil {
		return false, err
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == v1.PodPending || pod.Status.Phase == v1.PodRunning {
			return true, nil
		}
	}

	return false, nil
}

func (p *PeriklesHandler) podPartOfADeployment(pod *v1.Pod) (*appsv1.Deployment, error) {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			rs, err := p.Kube.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind == "Deployment" {
					return p.Kube.AppsV1().Deployments(pod.Namespace).Get(ctx, rsOwner.Name, metav1.GetOptions{})
				}
			}
		}
	}
	return nil, fmt.Errorf("no deployment found")
}

func (p *PeriklesHandler) podPartOfAJob(pod *v1.Pod) (*batchv1.Job, error) {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "Job" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return p.Kube.BatchV1().Jobs(pod.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
		}
	}
	return nil, fmt.Errorf("no job found")
}

func (p *PeriklesHandler) ensureSecrets(secretName, ns string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secret, err := p.Kube.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return fmt.Errorf("secret %s not found: %w", secretName, err)
	}

	requiredKeys := []string{"tls.key", "tls.pem", "tls.crt"}
	for _, key := range requiredKeys {
		if _, exists := secret.Data[key]; !exists {
			logging.Debug(fmt.Sprintf("Key %s missing in secret %s", key, secretName))
			return fmt.Errorf("missing key %s in secret %s", key, secretName)
		}
	}

	return nil
}

func (p *PeriklesHandler) cleanUpMapping(serviceToRemove string) error {
	mapping, err := p.Mapping.Get(p.CrdName)
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if err != nil {
		return err
	}

	newMapping := v1alpha.Mapping{}

	for _, service := range mapping.Spec.Services {
		if service.Name == serviceToRemove {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := p.Kube.CoreV1().Secrets(service.Namespace).Delete(ctx, service.SecretName, metav1.DeleteOptions{}); err != nil {
				if !errors.IsNotFound(err) {
					return fmt.Errorf("failed to delete secret %s: %w", service.SecretName, err)
				}
			}
			logging.Debug(fmt.Sprintf("Deleted secret %s for orphaned service %s", service.SecretName, service.Name))

			continue
		}

		newService := v1alpha.Service{
			Name:       service.Name,
			KubeType:   service.KubeType,
			SecretName: service.SecretName,
			Namespace:  service.Namespace,
			Active:     service.Active,
			Created:    service.Created,
			Validity:   service.Validity,
			Clients:    []v1alpha.Client{},
		}

		for _, client := range service.Clients {
			if client.Name == serviceToRemove {
				continue
			}

			addClient := true
			for _, newClient := range newService.Clients {
				if newClient.Name == client.Name {
					addClient = false
					break
				}
			}

			if addClient {
				newService.Clients = append(newService.Clients, client)
			}
		}

		newMapping.Spec.Services = append(newMapping.Spec.Services, newService)
	}

	mapping.Spec.Services = newMapping.Spec.Services
	_, err = p.Mapping.Update(mapping)

	logging.Debug(fmt.Sprintf("updated mapping after removing: %s", serviceToRemove))
	return err
}
