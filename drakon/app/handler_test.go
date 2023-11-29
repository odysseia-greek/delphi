package app

import (
	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/plato/config"
	configs "github.com/odysseia-greek/delphi/drakon/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandlerCreateDocuments(t *testing.T) {
	t.Run("CreateRole", func(t *testing.T) {
		file := "createRole"
		status := 200
		mockElasticClient, err := aristoteles.NewMockClient(file, status)
		assert.Nil(t, err)

		roles := []string{config.SeederElasticRole, config.HybridElasticRole, config.ApiElasticRole}

		for _, role := range roles {
			testConfig := configs.Config{
				Elastic: mockElasticClient,
				Indexes: []string{"test"},
				Roles:   []string{role},
			}

			testHandler := DrakonHandler{Config: &testConfig}
			created, err := testHandler.CreateRoles()
			assert.Nil(t, err)
			assert.True(t, created)
		}
	})

	t.Run("Failed", func(t *testing.T) {
		file := "createRole"
		status := 502
		mockElasticClient, err := aristoteles.NewMockClient(file, status)
		assert.Nil(t, err)

		testConfig := configs.Config{
			Elastic: mockElasticClient,
			Indexes: []string{"test"},
			Roles:   []string{"rike"},
		}

		testHandler := DrakonHandler{Config: &testConfig}
		created, err := testHandler.CreateRoles()
		assert.NotNil(t, err)
		assert.False(t, created)
	})
}
