package vault

import "github.com/odysseia-greek/agora/diogenes"

func (c *Client) CreateOneTimeToken(policies []string) (string, error) {
	return c.vault.CreateOneTimeToken(policies)
}

func (c *Client) CreateSecret(podName string, payload []byte) (bool, error) {
	return c.vault.CreateNewSecret(podName, payload)
}

func (c *Client) CreateElasticSecret(podName, username, password, elasticCert string) (bool, error) {
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

	return c.CreateSecret(podName, payload)
}
