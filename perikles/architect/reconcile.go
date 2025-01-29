package architect

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/thales/crd/v1alpha"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

// reconcileJob ensures resources are in the desired state and cleans up orphaned resources.
func (p *PeriklesHandler) reconcileJob() (err error) {
	// Defer panic recovery to avoid crashing the entire process
	defer func() {
		if r := recover(); r != nil {
			logging.Error(fmt.Sprintf("Recovered from panic in reconcileJob: %v", r))
			err = fmt.Errorf("panic occurred: %v", r)
		}
	}()

	mapping, err := p.Mapping.Get(p.CrdName)
	if err != nil {
		return fmt.Errorf("failed to get mapping: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	for _, service := range mapping.Spec.Services {
		if err := p.processService(ctx, service); err != nil {
			logging.Error(fmt.Sprintf("Failed to process service %s: %v", service.Name, err))
		}
	}

	return nil
}

// processService handles reconciliation for an individual service.
func (p *PeriklesHandler) processService(ctx context.Context, service v1alpha.Service) error {
	deploy, err := p.Kube.AppsV1().Deployments(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			if err := p.cleanUpOrphanedService(ctx, service); err != nil {
				return fmt.Errorf("failed to clean up orphaned service %s: %w", service.Name, err)
			}
			return nil
		}
		return fmt.Errorf("failed to get deployment %s: %w", service.Name, err)
	}

	// Check for associated clients
	for _, client := range service.Clients {
		if err := p.checkClientDeployment(ctx, client); err != nil {
			logging.Error(fmt.Sprintf("Failed to check client %s: %v", client.Name, err))
		}
	}

	// Ensure secrets are correctly configured
	if err := p.ensureSecrets(ctx, deploy, service); err != nil {
		logging.Error(fmt.Sprintf("Failed to ensure secrets for %s: %v", service.Name, err))
	}

	// Ensure network policies are correctly configured
	if err := p.ensureNetworkPolicies(ctx, deploy); err != nil {
		logging.Error(fmt.Sprintf("Failed to ensure network policies for %s: %v", service.Name, err))
	}

	return nil
}

func (p *PeriklesHandler) cleanUpOrphanedService(ctx context.Context, service v1alpha.Service) error {
	// Delete associated secret
	if err := p.Kube.CoreV1().Secrets(service.Namespace).Delete(ctx, service.SecretName, metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete secret %s: %w", service.SecretName, err)
		}
	}
	logging.Debug(fmt.Sprintf("Deleted secret %s for orphaned service %s", service.SecretName, service.Name))

	if err := p.cleanUpNetWorkPolicies(service.Name); err != nil {
		logging.Error(fmt.Sprintf("Failed to clean up network policies for %s: %v", service.Name, err))
	}
	// Clean up mapping
	if err := p.cleanUpMapping(service.Name); err != nil {
		return fmt.Errorf("failed to clean up mapping for service %s: %w", service.Name, err)
	}
	return nil
}

func (p *PeriklesHandler) checkClientDeployment(ctx context.Context, client v1alpha.Client) error {
	_, err := p.Kube.AppsV1().Deployments(client.Namespace).Get(ctx, client.Name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		if err := p.cleanUpMapping(client.Name); err != nil {
			return fmt.Errorf("failed to clean up mapping for client %s: %w", client.Name, err)
		}
	}
	return nil
}

func (p *PeriklesHandler) ensureSecrets(ctx context.Context, deploy *appsv1.Deployment, service v1alpha.Service) error {
	secretName := fmt.Sprintf("%s-tls-certs", deploy.Name)
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

func (p *PeriklesHandler) ensureNetworkPolicies(ctx context.Context, deploy *appsv1.Deployment) error {
	networkPolicies, err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(p.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list network policies: %w", err)
	}

	networkPoliciesRecreationTrigger := false

	// Parse deployment annotations
	for key, value := range deploy.Spec.Template.Annotations {
		switch key {
		case config.DefaultRoleAnnotation:
			found := false
			for _, networkpolicy := range networkPolicies.Items {
				if networkpolicy.Name == fmt.Sprintf("restrict-elasticsearch-access-%s", deploy.Name) {
					found = true
					break
				}
			}

			if !found {
				networkPoliciesRecreationTrigger = true
			}
		case AnnotationAccesses:
			accessList := strings.Split(value, ";")

			for _, client := range accessList {
				found := false
				for _, networkPolicy := range networkPolicies.Items {
					if strings.Contains(networkPolicy.Name, client) && !strings.Contains(networkPolicy.Name, "elastic") {
						found = true
						break
					}
				}

				if !found {
					networkPoliciesRecreationTrigger = true
				}
			}
		}
	}

	if networkPoliciesRecreationTrigger {
		err := p.checkForElasticAnnotations(deploy, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

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
		if strings.Contains(nwp.Name, serviceToRemove) {
			// Delete the matching network policy
			err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(p.Namespace).Delete(ctx, nwp.Name, metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("failed to delete network policy %s: %w", nwp.Name, err)
			}
			fmt.Printf("Deleted network policy: %s\n", nwp.Name)
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
	return err
}
