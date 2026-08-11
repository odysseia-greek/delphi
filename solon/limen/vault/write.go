package vault

import "context"

func (c *Client) WritePolicy(ctx context.Context, policyName string, policyRules []byte) error {
	return c.vault.WritePolicy(ctx, policyName, policyRules)
}
