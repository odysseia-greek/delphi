package lawgiver

import (
	"testing"

	"github.com/odysseia-greek/agora/diogenes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateVaultAuthConfig(t *testing.T) {
	t.Run("requires a dedicated projected token for Kubernetes auth", func(t *testing.T) {
		t.Setenv(diogenes.EnvAuthMethod, diogenes.AuthMethodKube)
		t.Setenv(diogenes.EnvVaultKubernetesTokenPath, "")

		err := validateVaultAuthConfig()

		require.Error(t, err)
		assert.Contains(t, err.Error(), diogenes.EnvVaultKubernetesTokenPath)
	})

	t.Run("accepts the projected Solon Vault token", func(t *testing.T) {
		t.Setenv(diogenes.EnvAuthMethod, diogenes.AuthMethodKube)
		t.Setenv(diogenes.EnvVaultKubernetesTokenPath, "/var/run/secrets/vault/token")

		assert.NoError(t, validateVaultAuthConfig())
	})

	t.Run("does not require a projected token for bootstrap token auth", func(t *testing.T) {
		t.Setenv(diogenes.EnvAuthMethod, diogenes.AuthMethodToken)
		t.Setenv(diogenes.EnvVaultKubernetesTokenPath, "")

		assert.NoError(t, validateVaultAuthConfig())
	})
}
