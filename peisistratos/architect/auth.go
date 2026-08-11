package architect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/odysseia-greek/agora/diogenes"
	plato "github.com/odysseia-greek/agora/plato/config"
)

const (
	defaultVaultService          = "https://vault:8200"
	defaultPeisistratosTokenPath = "/var/run/secrets/peisistratos-vault/token"
	envVaultKubernetesTokenPath  = "VAULT_KUBERNETES_TOKEN_PATH"
)

func (p *PeisistratosHandler) loginForReconciliation(ctx context.Context) error {
	tokenPath := plato.StringFromEnv(envVaultKubernetesTokenPath, defaultPeisistratosTokenPath)
	jwt, err := os.ReadFile(tokenPath)
	if err != nil {
		return fmt.Errorf("read Peisistratos projected Vault token: %w", err)
	}

	address := plato.StringFromEnv(diogenes.EnvVaultService, defaultVaultService)
	tlsConfig, err := vaultTLSConfig()
	if err != nil {
		return err
	}

	client, err := diogenes.CreateVaultClientKubernetes(ctx, address, peisistratosPolicyName, string(jwt), tlsConfig)
	if err != nil {
		return fmt.Errorf("authenticate Peisistratos with Vault Kubernetes auth: %w", err)
	}
	p.Vault = client
	return nil
}

func vaultTLSConfig() (*api.TLSConfig, error) {
	if !envEnabled(diogenes.EnvTLSEnabled) {
		return nil, nil
	}

	rootPath := plato.StringFromEnv(diogenes.EnvRootTlSDir, plato.DefaultTLSFileLocation)
	secretPath := filepath.Join(rootPath, "vault")
	ca, err := firstTLSFile(secretPath, "ca.crt", "vault.ca")
	if err != nil {
		return nil, err
	}
	cert, err := firstTLSFile(secretPath, "tls.crt", "vault.crt")
	if err != nil {
		return nil, err
	}
	key, err := firstTLSFile(secretPath, "tls.key", "vault.key")
	if err != nil {
		return nil, err
	}

	return diogenes.CreateTLSConfig(ca, cert, key, secretPath), nil
}

func firstTLSFile(directory string, names ...string) (string, error) {
	for _, name := range names {
		path := filepath.Join(directory, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("Vault TLS file not found in %s (tried %s)", directory, strings.Join(names, ", "))
}

func envEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
