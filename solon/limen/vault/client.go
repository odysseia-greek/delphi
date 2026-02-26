package vault

import "github.com/odysseia-greek/agora/diogenes"

type Client struct {
	vault diogenes.Client
}

func NewClient(vault diogenes.Client) *Client {
	return &Client{
		vault: vault,
	}
}

func (c *Client) Health() (bool, error) {
	return c.vault.Health()
}
