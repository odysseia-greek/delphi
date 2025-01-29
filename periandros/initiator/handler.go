package initiator

import (
	"encoding/json"
	"fmt"
	uuid2 "github.com/google/uuid"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/models"
	"github.com/odysseia-greek/agora/plato/service"
	"time"
)

type PeriandrosHandler struct {
	Duration             time.Duration
	Timeout              time.Duration
	Namespace            string
	HttpClients          service.OdysseiaClient
	SolonCreationRequest models.SolonCreationRequest
}

func (p *PeriandrosHandler) CreateUser() (bool, error) {
	healthy := p.CheckSolonHealth()
	if !healthy {
		return false, fmt.Errorf("solon not available cannot create user")
	}

	uuid := uuid2.New().String()

	response, err := p.HttpClients.Solon().Register(p.SolonCreationRequest, uuid)
	if err != nil {
		return false, err
	}

	defer response.Body.Close()

	var solonResponse models.SolonResponse
	err = json.NewDecoder(response.Body).Decode(&solonResponse)
	if err != nil {
		return false, err
	}

	return solonResponse.UserCreated, nil
}

func (p *PeriandrosHandler) CheckSolonHealth() bool {
	healthy := false

	ticker := time.NewTicker(p.Duration)
	timeout := time.After(p.Timeout)

	for {
		select {
		case t := <-ticker.C:
			logging.Info(fmt.Sprintf("tick: %s", t))

			uuid := uuid2.New().String()

			response, err := p.HttpClients.Solon().Health(uuid)
			if err != nil {
				logging.Error(fmt.Sprintf("Error getting response: %s", err))
				continue
			}

			var solonResponse models.Health
			err = json.NewDecoder(response.Body).Decode(&solonResponse)
			if err != nil {
				continue
			}

			healthy = solonResponse.Healthy
			if !healthy {
				continue
			}
			ticker.Stop()

		case <-timeout:
			ticker.Stop()
		}
		break
	}

	return healthy
}
