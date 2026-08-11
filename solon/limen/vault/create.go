package vault

import (
	"context"

	"github.com/odysseia-greek/agora/diogenes"
)

func (c *Client) CreateOneTimeToken(ctx context.Context, policies []string) (string, error) {
	return c.vault.CreateOneTimeToken(ctx, policies)
}

func (c *Client) CreateSecret(ctx context.Context, podName string, payload []byte) (bool, error) {
	return c.vault.CreateNewSecret(ctx, podName, payload)
}

func (c *Client) CreateElasticSecret(ctx context.Context, podName, username, password, elasticCert string) (bool, error) {
	createRequest := diogenes.CreateSecretRequest{
		Data: diogenes.ElasticConfigVault{
			Username:    username,
			Password:    password,
			ElasticCERT: elasticCert,
		},
	}

	payload, err := createRequest.Marshal()
	if err != nil {
		return false, err
	}

	return c.CreateSecret(ctx, podName, payload)
}
