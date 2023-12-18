package architect

import (
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	kubernetes "github.com/odysseia-greek/agora/thales"
)

func CreateNewConfig(env string) (*PeisistratosHandler, error) {
	healthCheck := true
	outOfClusterKube := false
	debugMode := false
	if env == "DEVELOPMENT" {
		healthCheck = false
		outOfClusterKube = true
		debugMode = true
	}

	vault, err := diogenes.CreateVaultClient(env, healthCheck, debugMode)
	if err != nil {
		return nil, err
	}

	podName := config.ParsedPodNameFromEnv()
	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	kube, err := kubernetes.CreateKubeClient(outOfClusterKube)
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
