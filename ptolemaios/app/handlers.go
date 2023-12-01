package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/models"
	"github.com/odysseia-greek/agora/plato/service"
	"github.com/odysseia-greek/delphi/ptolemaios/config"
	pb "github.com/odysseia-greek/delphi/ptolemaios/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
)

type PtolemaiosHandler struct {
	Config *config.Config
	pb.UnimplementedPtolemaiosServer
}

// GetSecret creates a 1 time token and returns the secret from vault
func (p *PtolemaiosHandler) GetSecret(ctx context.Context, unnamed *pb.VaultRequest) (*pb.ElasticConfigVault, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get metadata from context")
	}

	var traceID string
	headerValue := md.Get(service.HeaderKey)
	if len(headerValue) > 0 {
		traceID = headerValue[0]
	}

	oneTimeToken, err := p.getOneTimeToken(traceID)
	if err != nil {
		return nil, err
	}

	p.Config.Vault.SetOnetimeToken(oneTimeToken)
	secret, err := p.Config.Vault.GetSecret(p.Config.PodName)
	if err != nil {
		return nil, err
	}

	logging.Debug(fmt.Sprintf("found secret: %v", secret))

	if secret == nil {
		return nil, fmt.Errorf("secret came back empty")
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

	responseMd := metadata.New(map[string]string{service.HeaderKey: traceID})
	grpc.SendHeader(ctx, responseMd)

	return &elasticModel, nil
}

// GetSecretNamed creates a 1 time token and returns the secret from vault
func (p *PtolemaiosHandler) GetSecretNamed(ctx context.Context, named *pb.VaultRequestNamed) (*pb.ElasticConfigVault, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get metadata from context")
	}

	var traceID string
	headerValue := md.Get(service.HeaderKey)
	if len(headerValue) > 0 {
		traceID = headerValue[0]
	}

	oneTimeToken, err := p.getOneTimeToken(traceID)
	if err != nil {
		return nil, err
	}

	p.Config.Vault.SetOnetimeToken(oneTimeToken)
	secret, err := p.Config.Vault.GetSecret(named.PodName)
	if err != nil {
		return nil, err
	}

	logging.Debug(fmt.Sprintf("found secret: %v", secret))

	if secret == nil {
		return nil, fmt.Errorf("secret came back empty")
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

	responseMd := metadata.New(map[string]string{service.HeaderKey: traceID})
	grpc.SendHeader(ctx, responseMd)

	return &elasticModel, nil
}

func (p *PtolemaiosHandler) ShutDown(ctx context.Context, code *pb.ShutDownRequest) (*pb.ShutDownResponse, error) {
	logging.Debug(fmt.Sprintf("got code: %s", code))
	logging.Debug("Received shutdown request. Performing cleanup...")
	os.Exit(0)

	return &pb.ShutDownResponse{}, nil
}

func (p *PtolemaiosHandler) Health(context.Context, *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Health: true,
	}, nil
}

func (p *PtolemaiosHandler) getOneTimeToken(traceId string) (string, error) {
	response, err := p.Config.HttpClients.Solon().OneTimeToken(traceId)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	var tokenModel models.TokenResponse
	err = json.NewDecoder(response.Body).Decode(&tokenModel)
	if err != nil {
		return "", err
	}

	logging.Debug(fmt.Sprintf("found token: %s", tokenModel.Token))
	return tokenModel.Token, nil
}
