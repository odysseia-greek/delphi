package diplomat

import (
	"context"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/models"
	"github.com/odysseia-greek/agora/plato/service"
	pb "github.com/odysseia-greek/delphi/ptolemaios/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestGetOneTimeToken(t *testing.T) {
	scheme := "http"
	baseUrl := "somelocalhost.com"
	token := "s.49uwenfke9fue"
	uuid := "thisisnotauuid"
	tokenResponse := models.TokenResponse{Token: token}

	config := service.ClientConfig{
		Ca: nil,
		Solon: service.OdysseiaApi{
			Url:    baseUrl,
			Scheme: scheme,
			Cert:   nil,
		},
	}

	t.Run("Get", func(t *testing.T) {
		codes := []int{
			200,
		}

		r, err := tokenResponse.Marshal()
		assert.Nil(t, err)

		responses := []string{
			string(r),
		}

		testClient, err := service.NewFakeClient(config, codes, responses)
		assert.Nil(t, err)

		handler := AmbassadorServiceImpl{
			HttpClients: testClient,
		}

		sut, err := handler.getOneTimeToken(uuid)
		assert.Nil(t, err)
		assert.Equal(t, token, sut)
	})

	t.Run("GetWithError", func(t *testing.T) {
		codes := []int{
			500,
		}

		responses := []string{
			"error: You created",
		}

		testClient, err := service.NewFakeClient(config, codes, responses)
		assert.Nil(t, err)

		handler := AmbassadorServiceImpl{
			HttpClients: testClient,
		}

		sut, err := handler.getOneTimeToken(uuid)
		assert.NotNil(t, err)
		assert.Equal(t, "", sut)
	})
}

func TestHealthEndpoint(t *testing.T) {
	handler := AmbassadorServiceImpl{}

	t.Run("Health", func(t *testing.T) {
		sut, err := handler.Health(context.Background(), &pb.HealthRequest{})
		assert.Nil(t, err)
		assert.True(t, sut.Health)
	})
}

func TestGetSecrets(t *testing.T) {
	scheme := "http"
	baseUrl := "somelocalhost.com"
	token := "s.49uwenfke9fue"
	uuid := "thisisnotauuid"
	podName := "alexandros-api-202"
	tokenResponse := models.TokenResponse{Token: token}

	config := service.ClientConfig{
		Ca: nil,
		Solon: service.OdysseiaApi{
			Url:    baseUrl,
			Scheme: scheme,
			Cert:   nil,
		},
	}

	t.Run("GetNamed", func(t *testing.T) {
		codes := []int{
			200,
		}

		r, err := tokenResponse.Marshal()
		assert.Nil(t, err)

		responses := []string{
			string(r),
		}

		testClient, err := service.NewFakeClient(config, codes, responses)
		assert.Nil(t, err)

		fixtures := []string{"createSecret"}
		vaultClient, err := diogenes.CreateMockVaultClient(fixtures, 200)
		assert.Nil(t, err)

		handler := AmbassadorServiceImpl{
			HttpClients: testClient,
			Vault:       vaultClient,
		}

		req := &pb.VaultRequestNamed{PodName: podName}
		md := metadata.Pairs(service.HeaderKey, uuid)

		// Create a new context with the custom metadata
		ctx := metadata.NewOutgoingContext(context.Background(), md)

		sut, err := handler.GetNamedSecret(ctx, req)
		assert.Nil(t, err)
		assert.Equal(t, "", sut.ElasticUsername)
	})

	t.Run("GetUnnamed", func(t *testing.T) {
		codes := []int{
			200,
		}

		r, err := tokenResponse.Marshal()
		assert.Nil(t, err)

		responses := []string{
			string(r),
		}

		testClient, err := service.NewFakeClient(config, codes, responses)
		assert.Nil(t, err)

		fixtures := []string{"retrieveSecret"}
		vaultClient, err := diogenes.CreateMockVaultClient(fixtures, 200)
		assert.Nil(t, err)

		handler := AmbassadorServiceImpl{
			HttpClients: testClient,
			Vault:       vaultClient,
		}

		req := &pb.VaultRequest{}
		sut, err := handler.GetSecret(context.Background(), req)
		assert.Nil(t, err)
		assert.Equal(t, "", sut.ElasticUsername)
	})
}
