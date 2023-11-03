package app

import (
	"github.com/odysseia-greek/diogenes"
	"github.com/odysseia-greek/plato/config"
	kubernetes "github.com/odysseia-greek/thales"
)

func CreateNewConfig(env string) (*PeisistratosHandler, error) {
	healthCheck := true
	outOfClusterKube := false
	if env == "LOCAL" || env == "TEST" {
		healthCheck = false
		outOfClusterKube = true
	}

	vault, err := diogenes.CreateVaultClient(env, healthCheck)
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
		Namespace: ns,
		PodName:   podName,
		Vault:     vault,
		Threshold: 2,
		Shares:    5,
		Kube:      kube,
		Env:       env,
	}, nil
}
