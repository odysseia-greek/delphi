package architect

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/thales/crd/v1alpha"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

func (p *PeriklesHandler) cleanUpNetWorkPolicies(serviceToRemove string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// List all network policies in the namespace
	nwps, err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(p.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list network policies: %w", err)
	}

	// Iterate through the list and delete policies that match the service name
	for _, nwp := range nwps.Items {
		if strings.Contains(nwp.Name, "allow-all") {
			continue
		}
		if strings.Contains(nwp.Name, fmt.Sprintf("allow-%s-access", serviceToRemove)) || strings.Contains(nwp.Name, fmt.Sprintf("elasticsearch-access-%s", serviceToRemove)) {
			// Delete the matching network policy
			err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(p.Namespace).Delete(ctx, nwp.Name, metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("failed to delete network policy %s: %w", nwp.Name, err)
			}
			logging.Debug(fmt.Sprintf("Deleted network policy: %s", nwp.Name))
		}
	}

	return nil
}

func (p *PeriklesHandler) podPartOfADeployment(pod *v1.Pod) (*appsv1.Deployment, error) {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			rs, err := p.Kube.AppsV1().ReplicaSets(p.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind == "Deployment" {
					return p.Kube.AppsV1().Deployments(p.Namespace).Get(ctx, rsOwner.Name, metav1.GetOptions{})
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
			return p.Kube.BatchV1().Jobs(p.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
		}
	}
	return nil, fmt.Errorf("no job found")
}

func (p *PeriklesHandler) ensureSecrets(secretName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secret, err := p.Kube.CoreV1().Secrets(p.Namespace).Get(ctx, secretName, metav1.GetOptions{})
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
