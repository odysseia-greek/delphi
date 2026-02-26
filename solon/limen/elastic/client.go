package elastic

import (
	"github.com/odysseia-greek/agora/aristoteles"
	elasticmodels "github.com/odysseia-greek/agora/aristoteles/models"
)

type Client struct {
	elastic aristoteles.Client
}

func NewClient(elastic aristoteles.Client) *Client {
	return &Client{
		elastic: elastic,
	}
}

func (c *Client) HealthInfo() elasticmodels.DatabaseHealth {
	return c.elastic.Health().Info()
}
