package ktesias

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/service"
	"github.com/odysseia-greek/delphi/ptolemaios/diplomat"
	pb "github.com/odysseia-greek/delphi/ptolemaios/proto"
	"google.golang.org/grpc/metadata"
	"os"
	"strings"
	"time"
)

func (l *OdysseiaFixture) ptolemaiosIsAskedForTheCurrentConfig() error {
	ambassador := diplomat.NewClientAmbassador()

	healthy := ambassador.WaitForHealthyState()
	if !healthy {
		logging.Info("ptolemaios service not ready - restarting seems the only option")
		os.Exit(1)
	}

	traceId := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	md := metadata.New(map[string]string{service.HeaderKey: traceId})
	ctx = metadata.NewOutgoingContext(context.Background(), md)
	vaultConfig, err := ambassador.GetSecret(ctx, &pb.VaultRequest{})
	if err != nil {
		return err
	}

	l.ctx = context.WithValue(l.ctx, VaultConfig, vaultConfig)
	return nil
}

func (l *OdysseiaFixture) aShouldBeReturned(responseCode int) error {
	responseCodeFromCtx := l.ctx.Value(ElasticResponseCode).(int)
	if responseCodeFromCtx != responseCode {
		return fmt.Errorf("expected response code to be %d, got %d", responseCode, responseCodeFromCtx)
	}

	return nil
}

func (l *OdysseiaFixture) anElasticClientIsCreatedWithTheVaultData() error {
	vaultConfig := l.ctx.Value(VaultConfig).(*pb.ElasticConfigVault)
	tls := config.BoolFromEnv(config.EnvTlSKey)

	elasticService := aristoteles.ElasticService(tls)

	cfg := models.Config{
		Service:     elasticService,
		Username:    vaultConfig.ElasticUsername,
		Password:    vaultConfig.ElasticPassword,
		ElasticCERT: vaultConfig.ElasticCERT,
	}

	elastic, err := aristoteles.NewClient(cfg)
	if err != nil {
		return err
	}

	l.ctx = context.WithValue(l.ctx, ElasticClientContext, elastic)

	return nil
}

func (l *OdysseiaFixture) aCallIsMadeToAnIndexNotPartOfTheAnnotations() error {
	elasticClientLocal := l.ctx.Value(ElasticClientContext).(aristoteles.Client)

	query := elasticClientLocal.Builder().MatchAll()

	_, err := elasticClientLocal.Query().Match("thisisnotadrakoncreatedindex", query)
	if err != nil {
		if strings.Contains(err.Error(), "401") {
			l.ctx = context.WithValue(l.ctx, ElasticResponseCode, 401)
		} else if strings.Contains(err.Error(), "403") {
			l.ctx = context.WithValue(l.ctx, ElasticResponseCode, 403)
		} else {
			return err
		}
	}

	return nil
}

func (l *OdysseiaFixture) anElasticClientIsCreatedWithTheOneTimeTokenThatWasCreated() error {
	var oneTimeToken string
	if token, ok := l.ctx.Value(SecondTokenContext).(string); ok && token != "" {
		oneTimeToken = token
	} else if fallbackToken, ok := l.ctx.Value(TokenContext).(string); ok && fallbackToken != "" {
		oneTimeToken = fallbackToken
	} else {
		return fmt.Errorf("both SecondTokenContext and TokenContext are nil or empty")
	}

	l.Vault.SetOnetimeToken(oneTimeToken)
	secret, err := l.Vault.GetSecret(l.PodName)
	if err != nil {
		return err
	}

	var elasticModel pb.ElasticConfigVault
	for key, value := range secret.Data {
		if key == "data" {
			j, _ := json.Marshal(value)
			err := json.Unmarshal(j, &elasticModel)
			if err != nil {
				return err
			}
		}
	}

	elasticService := aristoteles.ElasticService(true)

	cfg := models.Config{
		Service:     elasticService,
		Username:    elasticModel.ElasticUsername,
		Password:    elasticModel.ElasticPassword,
		ElasticCERT: elasticModel.ElasticCERT,
	}

	elastic, err := aristoteles.NewClient(cfg)
	if err != nil {
		return err
	}

	l.ctx = context.WithValue(l.ctx, ElasticClientContext, elastic)

	return nil
}

func (l *OdysseiaFixture) aCallIsMadeToTheCorrectIndexWithTheCorrectAction() error {
	envAccess := config.SliceFromEnv(config.EnvIndex)[0]
	elasticClientLocal := l.ctx.Value(ElasticClientContext).(aristoteles.Client)

	query := elasticClientLocal.Builder().MatchAll()

	response, err := elasticClientLocal.Query().Match(envAccess, query)
	if err != nil {
		return err
	}

	if response != nil {
		l.ctx = context.WithValue(l.ctx, ElasticResponseCode, 200)
	}

	return nil
}

func int32Ptr(i int32) *int32 {
	return &i
}
