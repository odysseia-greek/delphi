package architect

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/perikles/pkg/service_mapping/crd/v1alpha"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *PeriklesHandler) cleanUpNetWorkPolicies(serviceToRemove, ns string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	allNamespaces := append([]string{p.Namespace}, p.WatchedNamespaces...)
	for _, namespace := range allNamespaces {
		nwpInWatchedNs, err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list network policies in %s namespace: %w", namespace, err)
		}

		for _, nwp := range nwpInWatchedNs.Items {
			if strings.Contains(nwp.Name, "allow-all") {
				continue
			}

			if strings.Contains(nwp.Name, fmt.Sprintf("allow-%s-access", serviceToRemove)) {
				err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(namespace).Delete(ctx, nwp.Name, metav1.DeleteOptions{})
				if err != nil {
					return fmt.Errorf("failed to delete network policy %s: %w", nwp.Name, err)
				}
				logging.Debug(fmt.Sprintf("Deleted network policy: %s in ns: %s", nwp.Name, namespace))
			}
		}
	}

	return nil
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
