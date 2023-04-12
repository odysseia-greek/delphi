package app

import (
	"context"
	"encoding/json"
	uuid2 "github.com/google/uuid"
	"github.com/kpango/glg"
	"github.com/odysseia-greek/delphi/ptolemaios/config"
	"github.com/odysseia-greek/plato/models"
	pb "github.com/odysseia-greek/plato/proto"
	"time"
)

type PtolemaiosHandler struct {
	Config   *config.Config
	Duration time.Duration
	pb.UnimplementedPtolemaiosServer
}

// GetSecret creates a 1 time token and returns the secret from vault
func (p *PtolemaiosHandler) GetSecret(context.Context, *pb.VaultRequest) (*pb.ElasticConfigVault, error) {
	uuid := uuid2.New().String()
	oneTimeToken, err := p.getOneTimeToken(uuid)
	if err != nil {
		return nil, err
	}

	glg.Debug("so far so good")
	p.Config.Vault.SetOnetimeToken(oneTimeToken)
	secret, err := p.Config.Vault.GetSecret(p.Config.PodName)
	if err != nil {
		return nil, err
	}

	var elasticModel pb.ElasticConfigVault
	for key, value := range secret.Data {
		if key == "data" {
			j, _ := json.Marshal(value)
			err := json.Unmarshal(j, &elasticModel)
			if err != nil {
				return nil, err
			}
		}
	}

	return &elasticModel, nil
}

func (p *PtolemaiosHandler) getOneTimeToken(uuid string) (string, error) {
	response, err := p.Config.HttpClients.Solon().OneTimeToken(uuid)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	var tokenModel models.TokenResponse
	err = json.NewDecoder(response.Body).Decode(&tokenModel)
	if err != nil {
		return "", err
	}

	glg.Debugf("found token: %s", tokenModel.Token)
	return tokenModel.Token, nil
}

func (p *PtolemaiosHandler) CheckForJobExit(exitChannel chan bool) {
	var counter int
	for {
		counter++
		glg.Debugf("run number: %d", counter)
		time.Sleep(p.Duration)
		pod, err := p.Config.Kube.Workload().GetPodByName(p.Config.Namespace, p.Config.FullPodName)
		if err != nil {
			glg.Errorf("error getting kube response %s", err)
			continue
		}

		for _, container := range pod.Status.ContainerStatuses {
			if container.Name == p.Config.PodName {
				glg.Debug(container.Name)
				if container.State.Terminated == nil {
					glg.Debugf("%s not done yet", container.Name)
					continue
				}
				if container.State.Terminated.ExitCode == 0 {
					exitChannel <- true
				}
			}
		}
	}
}
