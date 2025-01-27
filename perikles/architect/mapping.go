package architect

import (
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/thales/crd/v1alpha"
	"math/rand"
	"time"
)

const (
	timeFormat string = "2006-01-02 15:04:05"
)

func (p *PeriklesHandler) addHostToMapping(update MappingUpdate) error {
	return retry(3, 2*time.Second, func() error {
		p.Mutex.Lock()
		defer p.Mutex.Unlock()

		mapping, err := p.Mapping.Get(p.CrdName)
		if err != nil {
			return err
		}

		for i, service := range mapping.Spec.Services {
			if service.Name == update.HostName {
				service.Active = true
				service.Validity = update.Validity
				service.KubeType = update.KubeType
				service.SecretName = update.SecretName
				mapping.Spec.Services[i] = service
				_, err = p.Mapping.Update(mapping)
				return err
			}
		}

		service := v1alpha.Service{
			Name:       update.HostName,
			KubeType:   update.KubeType,
			Namespace:  p.Namespace,
			SecretName: update.SecretName,
			Active:     true,
			Validity:   update.Validity,
			Created:    time.Now().UTC().Format(timeFormat),
			Clients:    []v1alpha.Client{},
		}
		mapping.Spec.Services = append(mapping.Spec.Services, service)
		_, err = p.Mapping.Update(mapping)
		return err
	})
}

func (p *PeriklesHandler) checkMappingForUpdates() error {
	mapping, err := p.Mapping.Get(p.CrdName)
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

			go p.staggerRestarts(service)
		}
	}

	return nil
}

func (p *PeriklesHandler) staggerRestarts(service v1alpha.Service) {
	// after 20 minutes all hosts should be updated
	time.Sleep(20 * time.Minute)
	// all clients need to be restarted within an hour a staggered to avoid conflicts
	for _, client := range service.Clients {
		randomNumber := rand.Intn(120)
		time.Sleep(time.Duration(randomNumber) * time.Second)
		// there is some time between a secret update and that secret being updated in the running pod
		err := retry(20, 1*time.Second, func() error {
			return p.restartDeployment(client.Namespace, client.Name)
		})

		if err != nil {
			logging.Error(fmt.Sprintf("failed to restart deployment: %s", client.Name))
		}
	}
}

func (p *PeriklesHandler) processPendingUpdates() {
	p.Mutex.Lock()
	updates := p.PendingUpdates
	p.PendingUpdates = make(map[string][]MappingUpdate) // Clear pending updates
	p.Mutex.Unlock()

	if updates == nil {
		return
	}

	for hostName, updatesForHost := range updates {
		var hostAdded bool
		for _, update := range updatesForHost {
			if update.IsHostUpdate && !hostAdded {
				err := p.addHostToMapping(update)
				if err != nil {
					logging.Error(fmt.Sprintf("Failed to add host %s: %v", hostName, err))
				} else {
					hostAdded = true
				}
			} else {
				err := p.addClientToMapping(update)
				if err != nil {
					logging.Error(fmt.Sprintf("Failed to add client %s to host %s: %v", update.ClientName, update.HostName, err))
				}
			}
		}
	}
}

func (p *PeriklesHandler) addClientToPendingUpdates(hostName, clientName, kubeType, secretName string, validity int, isHostUpdate bool) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	if p.PendingUpdates == nil {
		p.PendingUpdates = make(map[string][]MappingUpdate)
	}

	p.PendingUpdates[hostName] = append(p.PendingUpdates[hostName], MappingUpdate{
		HostName:     hostName,
		ClientName:   clientName,
		KubeType:     kubeType,
		SecretName:   secretName,
		Validity:     validity,
		IsHostUpdate: isHostUpdate,
	})
}

func (p *PeriklesHandler) addClientToMapping(update MappingUpdate) error {
	return retry(3, 2*time.Second, func() error {
		p.Mutex.Lock()

		// Fetch the current mapping
		mapping, err := p.Mapping.Get(p.CrdName)
		if err != nil {
			p.Mutex.Unlock()
			return err
		}

		// Check if the host already exists
		var hostExists bool
		for _, service := range mapping.Spec.Services {
			if service.Name == update.HostName {
				hostExists = true
				break
			}
		}

		// If the host does not exist, unlock the mutex and add the host
		if !hostExists {
			p.Mutex.Unlock()
			err := p.addHostToMapping(update)
			if err != nil {
				return err
			}

			// Re-acquire the mutex to proceed with client addition
			p.Mutex.Lock()
			// Refresh the mapping after adding the host
			mapping, err = p.Mapping.Get(p.CrdName)
			if err != nil {
				p.Mutex.Unlock()
				return err
			}
		}

		// Add the client to the existing or newly created host
		for i, service := range mapping.Spec.Services {
			if service.Name == update.HostName {
				if update.ClientName != "" {
					exists := false
					for _, existingClient := range service.Clients {
						if existingClient.Name == update.ClientName {
							exists = true
							break
						}
					}
					if !exists {
						service.Clients = append(service.Clients, v1alpha.Client{
							Name:      update.ClientName,
							KubeType:  update.KubeType,
							Namespace: p.Namespace,
						})
					}
				}

				// Update the service entry
				mapping.Spec.Services[i] = service
				break
			}
		}

		// Update the mapping in the cluster
		_, err = p.Mapping.Update(mapping)
		p.Mutex.Unlock()
		return err
	})
}

func calculateTimeDifference(valid int, created string) (bool, error) {
	redeploy := false
	// validity is in days recalculate to hours
	validity := valid * 24
	validFrom, err := time.Parse(timeFormat, created)
	if err != nil {
		return false, err
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
