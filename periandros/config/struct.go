package config

import (
	"github.com/odysseia-greek/agora/plato/models"
	"github.com/odysseia-greek/agora/plato/service"
)

type Config struct {
	Namespace            string
	HttpClients          service.OdysseiaClient
	SolonCreationRequest models.SolonCreationRequest
}
