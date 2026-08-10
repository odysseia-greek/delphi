package elastic

import (
	"context"
	"fmt"
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
)

func (c *Client) DeleteOrphan(username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.elastic.Access().DeleteUser(ctx, username)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to delete orphaned user: %s, %s", username, err.Error()))
		return nil
	}

	logging.System(fmt.Sprintf("deleted orphan user in elastic: %s", username))
	return nil
}
