package diplomat

import (
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
)

func CreateNewConfig(env string) (*AmbassadorServiceImpl, error) {
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

	podName := config.StringFromEnv(config.EnvPodName, config.DefaultPodname)
	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	return &AmbassadorServiceImpl{
		HttpClients: http,
		Vault:       vault,
		PodName:     podName,
		Namespace:   ns,
	}, nil
}
