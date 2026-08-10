package diplomat

import (
	"context"

	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
)

func CreateNewConfig(ctx context.Context) (*AmbassadorServiceImpl, error) {
	http, err := config.CreateOdysseiaClient()
	if err != nil {
		return nil, err
	}

	vault, err := diogenes.CreateVaultClient(ctx, true)
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
