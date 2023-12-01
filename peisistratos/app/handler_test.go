package app

import (
	"github.com/hashicorp/vault/api"
	"github.com/odysseia-greek/agora/diogenes"
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestHandler(t *testing.T) {
	fakeKube := kubernetes.NewFakeKubeClient()

	t.Run("CreateConfigForGCP", func(t *testing.T) {
		ring := "TEST_RING"
		key := "TEST_KEY"
		location := "TEST_LOC"
		os.Setenv("KEY_RING", ring)
		os.Setenv("CRYPTO_KEY", key)
		os.Setenv("LOCATION", location)

		sut := createUnsealConfig("gcp").(GCPConfig)
		assert.Equal(t, ring, sut.KeyRing)
		assert.Equal(t, key, sut.CryptoKey)
		assert.Equal(t, location, sut.Location)

		os.Unsetenv("KEY_RING")
		os.Unsetenv("CRYPTO_KEY")
		os.Unsetenv("LOCATION")
	})
	t.Run("UnsealShamir", func(t *testing.T) {
		fixtures := []string{"unsealed"}
		fakeVault, err := diogenes.CreateMockVaultClient(fixtures, 200)
		assert.Nil(t, err)

		handler := PeisistratosHandler{
			UnsealMethod: "",
			Vault:        fakeVault,
			Kube:         fakeKube,
		}

		init := &api.InitResponse{Keys: []string{"test"}}
		err = handler.unsealVault(init)
		assert.Nil(t, err)
	})

	t.Run("UnsealGCP", func(t *testing.T) {
		fixtures := []string{"unsealed"}
		fakeVault, err := diogenes.CreateMockVaultClient(fixtures, 200)
		assert.Nil(t, err)

		handler := PeisistratosHandler{
			UnsealMethod: "gcp",
			Vault:        fakeVault,
			Kube:         fakeKube,
		}

		init := &api.InitResponse{Keys: []string{"test"}}
		err = handler.unsealVault(init)
		assert.Nil(t, err)
	})
}
