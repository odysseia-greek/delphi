package elastic

import (
	"context"
	"fmt"
	"time"

	elasticmodels "github.com/odysseia-greek/agora/aristoteles/models"
)

func (c *Client) CreateUser(username, password string, roles []string) (bool, error) {
	userRequest := elasticmodels.CreateUserRequest{
		Password: password,
		Roles:    roles,
		FullName: username,
		Email:    fmt.Sprintf("%s@odysseia-greek.com", username),
		Metadata: &elasticmodels.Metadata{Version: 1},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.elastic.Access().CreateUserWithContext(ctx, username, userRequest)
}
