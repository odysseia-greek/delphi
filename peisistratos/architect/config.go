package architect

import (
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	kubernetes "github.com/odysseia-greek/agora/thales"
	"os"
)

func CreateNewConfig() (*PeisistratosHandler, error) {
	env := os.Getenv("ENV")

	vault, err := diogenes.CreateVaultClient(true)
	if err != nil {
		return nil, err
	}

	podName := config.ParsedPodNameFromEnv()
	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	kube, err := kubernetes.CreateKubeClient(false)
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
