package app

import (
	"github.com/odysseia-greek/agora/plato/config"
	kubernetes "github.com/odysseia-greek/agora/thales"
)

func CreateNewConfig(env string) (*KleisthenesHandler, error) {
	outOfClusterKube := false
	if env == "DEVELOPMENT" {
		outOfClusterKube = true
	}

	kube, err := kubernetes.CreateKubeClient(outOfClusterKube)
	if err != nil {
		return nil, err
	}

	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)
	perikles := config.StringFromEnv("PERIKLES", "")

	return &KleisthenesHandler{
		Kube:            kube,
		namespace:       ns,
		periklesService: perikles,
	}, nil
}
