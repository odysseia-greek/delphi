package architect

import (
	"context"

	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"os"
)

func CreateNewConfig(ctx context.Context) (*PeisistratosHandler, error) {
	env := os.Getenv("ENV")

	vault, err := diogenes.CreateVaultClient(ctx, true)
	if err != nil {
		return nil, err
	}

	podName := config.ParsedPodNameFromEnv()
	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	kube, err := newKubeClient()
	if err != nil {
		return nil, err
	}

	return &PeisistratosHandler{
		Namespace:    ns,
		PodName:      podName,
		Shares:       5,
		Threshold:    2,
		Env:          env,
		Vault:        vault,
		Kube:         kube,
		UnsealMethod: "",
	}, nil
}
