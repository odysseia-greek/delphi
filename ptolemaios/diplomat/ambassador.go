package diplomat

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/models"
	"github.com/odysseia-greek/agora/plato/service"
	pb "github.com/odysseia-greek/delphi/ptolemaios/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
)

// GetSecret creates a 1 time token and returns the secret from vault
func (a *AmbassadorServiceImpl) GetSecret(ctx context.Context, request *pb.VaultRequest) (*pb.ElasticConfigVault, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var traceID string
	if ok {
		headerValue := md.Get(service.HeaderKey)
		if len(headerValue) > 0 {
			traceID = headerValue[0]
		}

		logging.Trace(fmt.Sprintf("found traceId: %s", traceID))
	}

	generatedToken, err := a.getOneTimeToken(traceID)
	if err != nil {
		logging.Error(err.Error())
		return nil, err
	}
	logging.Debug(fmt.Sprintf("setting token: %s", generatedToken))
	a.Vault.SetOnetimeToken(generatedToken)
	logging.Debug(fmt.Sprintf("gathering secret: %s", a.PodName))
	secret, err := a.Vault.GetSecret(a.PodName)
	if err != nil {
		return nil, err
	}

	logging.Debug(fmt.Sprintf("found secret with requestId: %v", secret.RequestID))

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

// GetNamedSecret creates a 1 time token and returns the secret from vault
func (a *AmbassadorServiceImpl) GetNamedSecret(ctx context.Context, request *pb.VaultRequestNamed) (*pb.ElasticConfigVault, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var traceID string
	if ok {
		headerValue := md.Get(service.HeaderKey)
		if len(headerValue) > 0 {
			traceID = headerValue[0]
		}

		logging.Trace(fmt.Sprintf("found traceId: %s", traceID))
	}

	logging.Debug("gathering one time token")
	oneTimeToken, err := a.getOneTimeToken(traceID)
	if err != nil {
		return nil, err
	}

	logging.Debug(fmt.Sprintf("one time token found: %s", oneTimeToken))

	a.Vault.SetOnetimeToken(oneTimeToken)

	logging.Debug(fmt.Sprintf("gathering secret: %s", request.PodName))
	secret, err := a.Vault.GetSecret(request.PodName)
	if err != nil {
		return nil, err
	}

	logging.Debug(fmt.Sprintf("found secret with requestId: %v", secret.RequestID))

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

func (a *AmbassadorServiceImpl) ShutDown(ctx context.Context, code *pb.ShutDownRequest) (*pb.ShutDownResponse, error) {
	logging.Debug(fmt.Sprintf("got code: %s", code))
	logging.Debug("Received shutdown request. Performing cleanup...")
	os.Exit(0)

	return &pb.ShutDownResponse{}, nil
}

func (a *AmbassadorServiceImpl) Health(context.Context, *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Health: true,
	}, nil
}

func (a *AmbassadorServiceImpl) getOneTimeToken(traceId string) (string, error) {
	response, err := a.HttpClients.Solon().OneTimeToken(traceId)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	var tokenModel models.TokenResponse
	err = json.NewDecoder(response.Body).Decode(&tokenModel)
	if err != nil {
		return "", err
	}

	logging.Debug(fmt.Sprintf("received token: %s", tokenModel.Token))
	return tokenModel.Token, nil
}
