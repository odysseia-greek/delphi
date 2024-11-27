package founder

import (
	"github.com/odysseia-greek/agora/plato/config"
	kubernetes "github.com/odysseia-greek/agora/thales"
)

func CreateNewConfig(env string) (*KleisthenesHandler, error) {
	kube, err := kubernetes.CreateKubeClient(false)
	if err != nil {
		return nil, err
	}

	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)
	perikles := config.StringFromEnv("PERIKLES", "perikles")

	return &KleisthenesHandler{
		Kube:            kube,
		namespace:       ns,
		periklesService: perikles,
	}, nil
}
