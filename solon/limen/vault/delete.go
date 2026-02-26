package vault

import (
	"fmt"

	"github.com/odysseia-greek/agora/plato/logging"
)

func (c *Client) DeleteOrphan(podName string) error {
	numberOfCleanedResource := 0
	err := c.vault.DeleteSecret(podName)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to delete orphaned secret: %s, %s", podName, err.Error()))
	}

	err = c.vault.RemoveSecret(podName)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to remove orphaned secret: %s, %s", podName, err.Error()))
	} else {
		logging.System(fmt.Sprintf("deleted orphan secret: %s", podName))
		numberOfCleanedResource++
	}

	policy := fmt.Sprintf("policy-%s", podName)

	deletedPolicy, err := c.vault.DeletePolicy(policy)
	if err != nil || deletedPolicy != nil {
		if err == nil {
			err = fmt.Errorf("unexpected delete policy response")
		}
		logging.Error(fmt.Sprintf("failed to delete orphaned policy: %s, %s", policy, err.Error()))
	} else {
		logging.System(fmt.Sprintf("deleted orphan policy: %s", policy))
		numberOfCleanedResource++
	}

	logging.System(fmt.Sprintf("finished cleanup service and cleaned up %d resources", numberOfCleanedResource))

	return nil
}
