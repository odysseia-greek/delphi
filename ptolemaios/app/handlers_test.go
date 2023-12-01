package app

import (
	"github.com/odysseia-greek/agora/plato/models"
	"github.com/odysseia-greek/agora/plato/service"
	configs "github.com/odysseia-greek/delphi/ptolemaios/config"
	"github.com/stretchr/testify/assert"
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

		testConfig := configs.Config{
			HttpClients: testClient,
		}

		handler := PtolemaiosHandler{Config: &testConfig}

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

		testConfig := configs.Config{
			HttpClients: testClient,
		}

		handler := PtolemaiosHandler{Config: &testConfig}

		sut, err := handler.getOneTimeToken(uuid)
		assert.NotNil(t, err)
		assert.Equal(t, "", sut)
	})
}
