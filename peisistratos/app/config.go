package app

import (
	"fmt"
	"github.com/odysseia-greek/diogenes"
	"github.com/odysseia-greek/plato/config"
	"github.com/odysseia-greek/plato/logging"
	kubernetes "github.com/odysseia-greek/thales"
	"os"
)

func CreateNewConfig(env string) (*PeisistratosHandler, error) {
	healthCheck := true
	outOfClusterKube := false
	debugMode := false
	if env == "LOCAL" || env == "TEST" {
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

	unsealMethod := os.Getenv("UNSEAL_PROVIDER")
	if unsealMethod != "" {
		logging.Debug(fmt.Sprintf("creating config with unseal provider: %s", unsealMethod))
	}

	return &PeisistratosHandler{
		Namespace:    ns,
		PodName:      podName,
		Shares:       5,
		Threshold:    2,
		Env:          env,
		UnsealMethod: unsealMethod,
		Vault:        vault,
		Kube:         kube,
	}, nil
}
