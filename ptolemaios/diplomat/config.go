package diplomat

import (
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
)

func CreateNewConfig(env string) (*AmbassadorServiceImpl, error) {
	http, err := config.CreateOdysseiaClient()
	if err != nil {
		return nil, err
	}

	vault, err := diogenes.CreateVaultClient(env, true, false)
	if err != nil {
		logging.Error(err.Error())
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
