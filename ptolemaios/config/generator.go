package config

import (
	"github.com/odysseia-greek/diogenes"
	"github.com/odysseia-greek/plato/config"
	"github.com/odysseia-greek/plato/service"
	kubernetes "github.com/odysseia-greek/thales"
)

type Config struct {
	HttpClients service.OdysseiaClient
	Vault       diogenes.Client
	Kube        kubernetes.KubeClient
	PodName     string
	Namespace   string
	RunOnce     bool
	FullPodName string
}

func CreateNewConfig(env string) (*Config, error) {
	healthCheck := true
	debugMode := false
	if env == "LOCAL" || env == "TEST" {
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

	kube, err := kubernetes.CreateKubeClient(healthCheck)
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
		Kube:        kube,
		PodName:     podName,
		Namespace:   ns,
		RunOnce:     runOnce,
		FullPodName: fullPodName,
	}, nil
}
