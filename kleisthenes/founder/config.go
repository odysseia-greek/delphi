package founder

import (
	"github.com/odysseia-greek/agora/plato/config"
)

func CreateNewConfig(env string) (*KleisthenesHandler, error) {
	kube, err := newKubeClient()
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
