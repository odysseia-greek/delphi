package legislator

import (
	"fmt"
	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
)

func CreateNewConfig() (*DrakonHandler, error) {
	tls := config.BoolFromEnv(config.EnvTlSKey)
	cfg, err := aristoteles.ElasticConfig(tls)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to create Elastic client operations will be interupted, %s", err.Error()))
	}

	elastic, err := aristoteles.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	err = aristoteles.HealthCheck(elastic)
	if err != nil {
		return nil, err
	}

	podName := config.ParsedPodNameFromEnv()
	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)
	roles := config.SliceFromEnv(config.EnvRoles)
	indexes := config.SliceFromEnv(config.EnvIndexes)

	return &DrakonHandler{
		Namespace: ns,
		PodName:   podName,
		Elastic:   elastic,
		Roles:     roles,
		Indexes:   indexes,
	}, nil
}
