package config

import (
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/service"
)

type Config struct {
	HttpClients service.OdysseiaClient
	Vault       diogenes.Client
	PodName     string
	Namespace   string
	RunOnce     bool
	FullPodName string
}

func CreateNewConfig(env string) (*Config, error) {
	healthCheck := true
	debugMode := false
	if env == "DEVELOPMENT" {
		healthCheck = false
		debugMode = true
	}

	http, err := config.CreateOdysseiaClient()
	if err != nil {
		return nil, err
	}

	vault, err := diogenes.CreateVaultClient(env, healthCheck, debugMode)
	if err != nil {
		return nil, err
	}

	podName := config.ParsedPodNameFromEnv()
	fullPodName := config.StringFromEnv(config.EnvPodName, config.DefaultPodname)
	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)
	runOnce := config.BoolFromEnv(config.EnvRunOnce)

	return &Config{
		HttpClients: http,
		Vault:       vault,
		PodName:     podName,
		Namespace:   ns,
		RunOnce:     runOnce,
		FullPodName: fullPodName,
	}, nil
}
