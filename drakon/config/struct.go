package config

import (
	"github.com/odysseia-greek/agora/aristoteles"
)

type Config struct {
	Namespace string
	PodName   string
	Elastic   aristoteles.Client
	Roles     []string
	Indexes   []string
}
