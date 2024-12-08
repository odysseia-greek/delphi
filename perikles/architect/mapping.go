package architect

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/thales/crd/v1alpha"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	timeFormat string = "2006-01-02 15:04:05"
)

func (p *PeriklesHandler) addHostToMapping(serviceName, secretName, kubeType string, validity int) (*v1alpha.Mapping, error) {
	var updatedMapping *v1alpha.Mapping

	err := retry(3, 2*time.Second, func() error {
		p.Config.Mutex.Lock()
		defer p.Config.Mutex.Unlock()

		mapping, err := p.Config.Mapping.Get(p.Config.CrdName)
		if err != nil {
			return err
		}

		for i, service := range mapping.Spec.Services {
			if service.Name == serviceName {
				service.Active = true
				service.Validity = validity
				service.KubeType = kubeType
				service.SecretName = secretName
				mapping.Spec.Services[i] = service
				updatedMapping, err = p.Config.Mapping.Update(mapping)
				if err != nil {
					return err
				}
				return nil
			}
		}

		service := v1alpha.Service{
			Name:       serviceName,
			KubeType:   kubeType,
			Namespace:  p.Config.Namespace,
			SecretName: secretName,
			Active:     true,
			Validity:   validity,
			Created:    time.Now().UTC().Format(timeFormat),
			Clients:    []v1alpha.Client{},
		}
		mapping.Spec.Services = append(mapping.Spec.Services, service)
		updatedMapping, err = p.Config.Mapping.Update(mapping)
		return err
	})

	if err != nil {
		return nil, err
	}

	return updatedMapping, nil
}

func (p *PeriklesHandler) addClientToMapping(hostName, clientName, kubeType string) (*v1alpha.Mapping, error) {
	var updatedMapping *v1alpha.Mapping

	err := retry(3, 2*time.Second, func() error {
		p.Config.Mutex.Lock()
		defer p.Config.Mutex.Unlock()

		mapping, err := p.Config.Mapping.Get(p.Config.CrdName)
		if err != nil {
			return err
		}

		client := v1alpha.Client{
			Name:      clientName,
			KubeType:  kubeType,
			Namespace: p.Config.Namespace,
		}

		for i, service := range mapping.Spec.Services {
			if service.Name == hostName {
				for _, existingClient := range service.Clients {
					if existingClient.Name == clientName {
						return nil // Already exists
					}
				}
				mapping.Spec.Services[i].Clients = append(mapping.Spec.Services[i].Clients, client)
			}
		}

		updatedMapping, err = p.Config.Mapping.Update(mapping)
		return err
	})

	if err != nil {
		return nil, err
	}

	return updatedMapping, nil
}

func (p *PeriklesHandler) loopForMappingUpdates() {
	ticker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-ticker.C:
			err := p.checkMappingForUpdates()
			if err != nil {
				logging.Error(err.Error())
			}
		}
	}
}

func (p *PeriklesHandler) cleanUpMapping(serviceToRemove string) error {
	mapping, err := p.Config.Mapping.Get(p.Config.CrdName)
	if err != nil {
		return err
	}

	if len(mapping.Spec.Services) == 0 {
		logging.Debug("service mapping empty no action required")
		return nil
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
	_, err = p.Config.Mapping.Update(mapping)
	return err

}

func (p *PeriklesHandler) checkMappingForUpdates() error {
	mapping, err := p.Config.Mapping.Get(p.Config.CrdName)
	if err != nil {
		return err
	}

	if len(mapping.Spec.Services) == 0 {
		logging.Debug("service mapping empty no action required")
		return nil
	}

	for _, service := range mapping.Spec.Services {
		redeploy, err := calculateTimeDifference(service.Validity, service.Created)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = p.Config.Kube.AppsV1().Deployments(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
		if err != nil {
			// If deployment doesn't exist, delete the associated secret
			if errors.IsNotFound(err) {
				err = p.Config.Kube.CoreV1().Secrets(service.Namespace).Delete(ctx, service.SecretName, metav1.DeleteOptions{})
				if err != nil {
					logging.Error(err.Error())
				} else {
					logging.Debug(fmt.Sprintf("deleted secret: %s for orphaned service: %s", service.SecretName, service.Name))
					err = p.cleanUpMapping(service.Name)
					if err != nil {
						logging.Error(err.Error())
					}
					return nil
				}
			}
		}

		for _, client := range service.Clients {
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err = p.Config.Kube.AppsV1().Deployments(client.Namespace).Get(ctx, client.Name, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					err = p.cleanUpMapping(client.Name)
					if err != nil {
						logging.Error(err.Error())
					}
				}
			}
		}

		if redeploy {
			logging.Debug(fmt.Sprintf("redeploy needed for service: %s", service.Name))
			logging.Debug("creating new certs after validity ran out")
			orgName := service.Namespace
			hostName := service.Name

			hosts := []string{
				fmt.Sprintf("%s", hostName),
				fmt.Sprintf("%s.%s", hostName, orgName),
				fmt.Sprintf("%s.%s.svc", hostName, orgName),
				fmt.Sprintf("%s.%s.svc.cluster.local", hostName, orgName),
			}
			err = p.createCert(hosts, service.Validity, service.SecretName)
			if err != nil {
				return err
			}

			// all clients need to be restarted within an hour a staggered to avoid conflicts
			for _, client := range service.Clients {
				// there is some time between a secret update and that secret being updated in the running pod
				err := retry(20, 1*time.Second, func() error {
					return p.restartDeployment(client.Namespace, client.Name)
				})

				if err != nil {
					logging.Error(fmt.Sprintf("failed to restart deployment: %s", client.Name))
				}
			}
		}
	}

	return nil
}

func calculateTimeDifference(valid int, created string) (bool, error) {
	redeploy := false
	// validity is in days recalculate to hours
	validity := valid * 24
	validFrom, err := time.Parse(timeFormat, created)
	if err != nil {
		return redeploy, err
	}

	inHours := time.Duration(validity) * time.Hour
	validTo := validFrom.Add(inHours)
	now := time.Now().UTC()

	timeDifference := validTo.Sub(now).Hours()

	if timeDifference < (time.Hour * 24).Hours() {
		redeploy = true
	}

	return redeploy, nil
}

func retry(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}
