package vault

func (c *Client) WritePolicy(policyName string, policyRules []byte) error {
	return c.vault.WritePolicy(policyName, policyRules)
}
